package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WOFClone struct {
	Count          int
	Success        int
	Error          int
	Skipped        int
	Source         string
	Dest           string
	Client         *http.Client
	connections    int64
	maxconnections int64
	cond           *sync.Cond
}

func NewWOFClone(source string, dest string) *WOFClone {

	// to do - add logging

	cd := &sync.Cond{L: &sync.Mutex{}}

	cl := &http.Client{}

	c := WOFClone{
		Count:          0,
		Success:        0,
		Error:          0,
		Skipped:        0,
		Source:         source,
		Dest:           dest,
		Client:         cl,
		connections:    0,
		maxconnections: 200,
		cond:           cd,
	}

	return &c
}

func (c *WOFClone) ParseMetaFile(file string) error {

	abs_path, _ := filepath.Abs(file)
	reader, read_err := csv.NewDictReader(abs_path)

	if read_err != nil {
		log.Println(read_err)
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
			c.PreProcess(rel_path)
		}()
	}

	wg.Wait()

	return nil
}

func (c *WOFClone) PreProcess(rel_path string) error {

	c.Count += 1

	remote := c.Source + rel_path
	local := path.Join(c.Dest, rel_path)

	_, err := os.Stat(local)

	if !os.IsNotExist(err) {

		change, _ := c.HasChanged(local, remote)

		if !change {
			return nil
		}

	}

	c.Process(remote, local)

	return nil
}

func (c *WOFClone) HasChanged(local string, remote string) (bool, error) {

	change := true

	body, err := ioutil.ReadFile(local)

	if err != nil {
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

	local_root := path.Dir(local)

	_, err := os.Stat(local_root)

	if os.IsNotExist(err) {
		log.Printf("create %s\n", local_root)
		os.MkdirAll(local_root, 0755)
	}

	log.Printf("fetch '%s' and store in %s\n", remote, local)

	rsp, fetch_err := c.Fetch("GET", remote)

	if fetch_err != nil {
		log.Println(fetch_err)
		return fetch_err
	}

	defer rsp.Body.Close()

	contents, read_err := ioutil.ReadAll(rsp.Body)

	if read_err != nil {
		log.Println(read_err)
		return read_err
	}

	go func() {
		write_err := ioutil.WriteFile(local, contents, 0644)

		if write_err != nil {
			log.Println(write_err)
		}
	}()

	return nil
}

func (c *WOFClone) Fetch(method string, url string) (*http.Response, error) {

	for {
		c.cond.L.Lock()

		for c.connections >= c.maxconnections {
			c.cond.Wait()
		}

		atomic.AddInt64(&c.connections, 1)	

		c.cond.L.Unlock()
		c.cond.Signal()
		break
	}

	req, _ := http.NewRequest(method, url, nil)
	req.Close = true

	rsp, err := c.Client.Do(req)

	atomic.AddInt64(&c.connections, -1)

	return rsp, err
}

func main() {

	var source = flag.String("source", "http://whosonfirst.mapzen.com/data/", "Where to look for files")
	var dest = flag.String("dest", "", "Where to write files")

	flag.Parse()
	args := flag.Args()

	cl := NewWOFClone(*source, *dest)

	start := time.Now()

	wg := new(sync.WaitGroup)

	for _, file := range args {

		wg.Add(1)

		go func() {
			defer wg.Done()
			log.Println(file)
			cl.ParseMetaFile(file)
		}()
	}

	wg.Wait()

	since := time.Since(start)
	secs := float64(since) / 1e9

	log.Printf("processed %d files in %f seconds\n", cl.Count, secs)
}
