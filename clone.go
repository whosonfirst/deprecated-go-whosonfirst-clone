package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	log "github.com/whosonfirst/go-whosonfirst-log"
	"github.com/jeffail/tunny"
	golog "log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WOFClone struct {
	Source         string
	Dest           string
	Count          int64
	Success        int64
	Error          int64
	Skipped        int64
	Scheduled      int64
	Completed      int64
	client         *http.Client
	connections    int64
	maxconnections int64
	cond           *sync.Cond
	logger         *log.WOFLogger
	pool	       *tunny.WorkPool
}

func NewWOFClone(source string, dest string, logger *log.WOFLogger) *WOFClone {

	cd := &sync.Cond{L: &sync.Mutex{}}

	cl := &http.Client{}

	numCPUs := runtime.NumCPU() * 4
	runtime.GOMAXPROCS(numCPUs)

	pool, _ := tunny.CreatePoolGeneric(numCPUs).Open()

	c := WOFClone{
		Count:          0,
		Success:        0,
		Error:          0,
		Skipped:        0,
		Source:         source,
		Dest:           dest,
		logger:         logger,
		client:         cl,
		connections:    0,
		maxconnections: 200,
		cond:           cd,
		pool:		pool,
	}

	return &c
}

func (c *WOFClone) ParseMetaFile(file string) error {

	abs_path, _ := filepath.Abs(file)
	c.logger.Debug("Parse meta file %s", abs_path)

	reader, read_err := csv.NewDictReader(abs_path)

	if read_err != nil {
		c.logger.Error("Failed to read %s, because %v", abs_path, read_err)
		return read_err
	}

	wg := new(sync.WaitGroup)

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		rel_path, ok := row["path"]

		if !ok {
			continue
		}

		wg.Add(1)

		go func() {

		   defer wg.Done()

		   complete, _ := c.Clone(rel_path)
		   c.logger.Debug("Finised cloning %s: %t", rel_path, complete)
		}()

		wg.Wait()
	}

	return nil
}

func (c *WOFClone) Clone(rel_path string) (bool, error) {

        defer c.pool.Close()

	c.logger.Debug("Pre-process %s", rel_path)

	atomic.AddInt64(&c.Count, 1)

	remote := c.Source + rel_path
	local := path.Join(c.Dest, rel_path)

	_, err := os.Stat(local)

	if !os.IsNotExist(err) {

		change, _ := c.HasChanged(local, remote)

		if !change {

			c.logger.Debug("%s has not changed so skipping", local)

			atomic.AddInt64(&c.Success, 1)
			atomic.AddInt64(&c.Skipped, 1)
			return true, nil
		}

	}

	_, err = c.pool.SendWork(func(){

	       c.logger.Debug("WORKING")

		atomic.AddInt64(&c.Scheduled, 1)

		process_err := c.Process(remote, local)

		atomic.AddInt64(&c.Completed, 1)

		if process_err != nil {
		   atomic.AddInt64(&c.Error, 1)
		} else {
		   atomic.AddInt64(&c.Success, 1)
		}
	})		
	
	return true, nil
}

/*
Okay, I figured out the problem I was seeing yesterday when checking
ETags (now that the WOF meta files contain an MD5 hash of the records
they list).

For example:

$> md5 /usr/local/mapzen/whosonfirst-data/data/404/529/391/404529391.geojson
71714a9cb8d286eb50b9a300b7faa1c3

If I request the same file from AWS proper everything is All Good (tm) :

$> curl -v https://s3.amazonaws.com/whosonfirst.mapzen.com/data/404/529/391/404529391.geojson
> /dev/null
* Server certificate: s3.amazonaws.com
< ETag: "71714a9cb8d286eb50b9a300b7faa1c3"

OTOH if I request the same file from wof.mz.com/data I get this:

$> curl -v https://whosonfirst.mapzen.com/data/404/529/391/404529391.geojson
> /dev/null
* Server certificate: whosonfirst.mapzen.com
< ETag: "93ad7d36a39f0e98f4aa777f22c73c03193fe390"

Which seems all weird-and-wtf until you remember that the nginx config
for the spelunker tries to proxy files from the wof-data GH repository
first and fall backs on AWS as a last resort:

https://github.com/whosonfirst/whosonfirst-www-spelunker/blob/master/nginx/whosonfirst-www-spelunker.conf.example

Who knows how GH is setting their ETags. I guess I don't really
understand why they aren't just doing MD5 hashes too but it's their
party so...

So.

The short answer is to also do a HEAD / ETag check against:

https://s3.amazonaws.com/whosonfirst.mapzen.com/data/

(20151027/thisisaaronland)
*/

func (c *WOFClone) HasChanged(local string, remote string) (bool, error) {

	change := true
	return change, nil

	c.Throttle()

	body, err := ioutil.ReadFile(local)

	atomic.AddInt64(&c.connections, -1)

	if err != nil {
		c.logger.Error("Failed to read %s, becase %v", local, err)
		golog.Fatal(err)
		return change, err
	}

	hash := md5.Sum(body)
	local_hash := hex.EncodeToString(hash[:])

	rsp, err := c.Fetch("HEAD", remote)

	if err != nil {
		return change, err
	}

	defer rsp.Body.Close()

	etag := rsp.Header.Get("Etag")
	remote_hash := strings.Replace(etag, "\"", "", -1)

	if local_hash == remote_hash {
		change = false
	}

	// log.Printf("hash %s etag %s change %t\n", local_hash, remote_hash, change)

	return change, nil
}

func (c *WOFClone) Process(remote string, local string) error {

	c.logger.Debug("fetch %s and store in %s", remote, local)

	local_root := path.Dir(local)

	_, err := os.Stat(local_root)

	if os.IsNotExist(err) {
		c.logger.Info("create %s", local_root)
		os.MkdirAll(local_root, 0755)
	}

	rsp, fetch_err := c.Fetch("GET", remote)

	if fetch_err != nil {
		return fetch_err
	}

	defer rsp.Body.Close()

	contents, read_err := ioutil.ReadAll(rsp.Body)

	if read_err != nil {
		c.logger.Error("failed to read body for %s, because %v", remote, read_err)
		return read_err
	}

	go func() error {

		write_err := ioutil.WriteFile(local, contents, 0644)

		if write_err != nil {
			c.logger.Error("Failed to write %s, because %v", local, write_err)

			// See the way we're in a goroutine? That means the parent function
			// will return 'okay' without an error. So we're just going to account
			// for that here...

			atomic.AddInt64(&c.Success, -1)
			atomic.AddInt64(&c.Error, 1)

			return write_err
		}

		c.logger.Debug("Wrote %s to disk", local)
		return nil
	}()

	return nil
}

func (c *WOFClone) Fetch(method string, url string) (*http.Response, error) {

	c.logger.Debug("%s %s", method, url)

	c.Throttle()

	req, _ := http.NewRequest(method, url, nil)
	req.Close = true

	rsp, err := c.client.Do(req)

	atomic.AddInt64(&c.connections, -1)

	if err != nil {
		c.logger.Error("Failed to %s %s, because %v", method, url, err)
		golog.Fatal(err)
		return nil, err
	}

	return rsp, err
}

func (c *WOFClone) Throttle() {

	for {
		c.cond.L.Lock()

		for c.connections > c.maxconnections {
			c.logger.Debug("%d still > %d", c.connections, c.maxconnections)
			c.cond.Wait()
		}

		atomic.AddInt64(&c.connections, 1)

		c.cond.L.Unlock()
		c.cond.Broadcast()
		break
	}

}

func (c *WOFClone) Monitor () {

	go func() {

		for c.Completed < c.Scheduled {
			c.logger.Info("scheduled: %d completed: %d connections: %d", c.Scheduled, c.Completed, c.connections)
			time.Sleep(5 * time.Second)
		}
	}()
}

func main() {

	// See notes inre source and Etags in the `HasChanged` method (20151027/thisisaaronland)

	var source = flag.String("source", "https://s3.amazonaws.com/whosonfirst.mapzen.com/data/", "Where to look for files")
	var dest = flag.String("dest", "", "Where to write files")
	var loglevel = flag.String("loglevel", "debug", "The level of detail for logging")

	flag.Parse()
	args := flag.Args()

	writer := io.MultiWriter(os.Stdout)

	lg := log.NewWOFLogger(writer, "[clone] ", *loglevel)

	cl := NewWOFClone(*source, *dest, lg)

	start := time.Now()

	wg := new(sync.WaitGroup)

	for _, file := range args {

		wg.Add(1)

		go func() {
			defer wg.Done()
			cl.ParseMetaFile(file)
		}()
	}

	cl.Monitor()

	wg.Wait()

	since := time.Since(start)
	secs := float64(since) / 1e9

	cl.logger.Info("processed %d files (ok: %d error: %d skipped: %d) in %f seconds\n", cl.Count, cl.Success, cl.Error, cl.Skipped, secs)
}
