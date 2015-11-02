package clone

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/jeffail/tunny"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	log "github.com/whosonfirst/go-whosonfirst-log"
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
)

type WOFClone struct {
	Source    string
	Dest      string
	Count     int64
	Success   int64
	Error     int64
	Skipped   int64
	Scheduled int64
	Completed int64
	Logger    *log.WOFLogger
	client    *http.Client
	pool      *tunny.WorkPool
}

func NewWOFClone(source string, dest string, procs int, logger *log.WOFLogger) *WOFClone {

	// cd := &sync.Cond{L: &sync.Mutex{}}

	cl := &http.Client{}

	runtime.GOMAXPROCS(procs)

	pool, _ := tunny.CreatePoolGeneric(procs).Open()

	c := WOFClone{
		Count:   0,
		Success: 0,
		Error:   0,
		Skipped: 0,
		Source:  source,
		Dest:    dest,
		Logger:  logger,
		client:  cl,
		pool:    pool,
	}

	return &c
}

func (c *WOFClone) CloneMetaFile(file string) error {

	abs_path, _ := filepath.Abs(file)
	// c.Logger.Debug("Parse meta file %s", abs_path)

	reader, read_err := csv.NewDictReader(abs_path)

	if read_err != nil {
		c.Logger.Error("Failed to read %s, because %v", abs_path, read_err)
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

		ensure_changes := true

		/*
			file_hash, ok := row["file_hash"]

			if ok {
			   remote := c.Source + rel_path	// sudo put me in a method
			   change, _ := c.HasHashChanged(file_hash, remote)

			   if !change{
			      c.logger.Info("no changes to %s", file_hash)
			      ensure_changes = false
			   }
			}
		*/

		wg.Add(1)
		atomic.AddInt64(&c.Scheduled, 1)

		go func() {

			defer wg.Done()

			_, err = c.pool.SendWork(func() {

				cl_err := c.ClonePath(rel_path, ensure_changes)

				if cl_err != nil {
					atomic.AddInt64(&c.Error, 1)
				} else {
					atomic.AddInt64(&c.Success, 1)
				}

				atomic.AddInt64(&c.Completed, 1)
				c.Status()
			})

		}()
	}

	wg.Wait()
	return nil
}

func (c *WOFClone) ClonePath(rel_path string, ensure_changes bool) error {

	atomic.AddInt64(&c.Count, 1)

	remote := c.Source + rel_path
	local := path.Join(c.Dest, rel_path)

	_, err := os.Stat(local)

	if !os.IsNotExist(err) && ensure_changes {

		change, _ := c.HasChanged(local, remote)

		if !change {

			c.Logger.Debug("%s has not changed so skipping", local)
			atomic.AddInt64(&c.Skipped, 1)
			return nil
		}

	}

	process_err := c.Process(remote, local)

	if process_err != nil {
		atomic.AddInt64(&c.Error, 1)
		return process_err
	}

	return nil
}

// don't return true if there's a problem - move that logic up above

func (c *WOFClone) HasChanged(local string, remote string) (bool, error) {

	change := true

	body, err := ioutil.ReadFile(local)

	if err != nil {
		c.Logger.Error("Failed to read %s, becase %v", local, err)
		return change, err
	}

	hash := md5.Sum(body)
	local_hash := hex.EncodeToString(hash[:])

	return c.HasHashChanged(local_hash, remote)
}

func (c *WOFClone) HasHashChanged(local_hash string, remote string) (bool, error) {

	change := true

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

	return change, nil
}

func (c *WOFClone) Process(remote string, local string) error {

	c.Logger.Debug("fetch %s and store in %s", remote, local)

	local_root := path.Dir(local)

	_, err := os.Stat(local_root)

	if os.IsNotExist(err) {
		c.Logger.Info("create %s", local_root)
		os.MkdirAll(local_root, 0755)
	}

	rsp, fetch_err := c.Fetch("GET", remote)

	if fetch_err != nil {
		return fetch_err
	}

	defer rsp.Body.Close()

	contents, read_err := ioutil.ReadAll(rsp.Body)

	if read_err != nil {
		c.Logger.Error("failed to read body for %s, because %v", remote, read_err)
		return read_err
	}

	go func() error {

		write_err := ioutil.WriteFile(local, contents, 0644)

		if write_err != nil {
			c.Logger.Error("Failed to write %s, because %v", local, write_err)

			atomic.AddInt64(&c.Success, -1)
			atomic.AddInt64(&c.Error, 1)

			return write_err
		}

		c.Logger.Debug("Wrote %s to disk", local)
		return nil
	}()

	return nil
}

func (c *WOFClone) Fetch(method string, url string) (*http.Response, error) {

	c.Logger.Debug("%s %s", method, url)

	req, _ := http.NewRequest(method, url, nil)
	req.Close = true

	rsp, err := c.client.Do(req)

	if err != nil {
		c.Logger.Error("Failed to %s %s, because %v", method, url, err)
		// golog.Fatal(err)
		return nil, err
	}

	return rsp, err
}

func (c *WOFClone) Status() {
	c.Logger.Info("scheduled: %d completed: %d success: %d error: %d goroutines: %d", c.Scheduled, c.Completed, c.Success, c.Error, runtime.NumGoroutine())
}
