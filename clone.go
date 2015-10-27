package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	Connections    int
	MaxConnections int
}

func NewWOFClone(source string, dest string) *WOFClone {

	// to do - add logging

	dl := net.Dialer{
		Timeout:   0,
		KeepAlive: 0,
	}

	tr := &http.Transport{
		Dial: dl.Dial,
	}

	cl := &http.Client{
	   Transport: tr,
	}

	c := WOFClone{
		Count:          0,
		Success:        0,
		Error:          0,
		Skipped:        0,
		Source:         source,
		Dest:           dest,
		Client:         cl,
		Connections:    0,
		MaxConnections: 200,
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

		for c.Connections > c.MaxConnections {

		     	 if c.Connections < c.MaxConnections {
			    log.Println("go")
	    		    break
	 		 }
     		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			c.FetchStore(rel_path)
		}()
	}

	wg.Wait()

	return nil
}

func (c *WOFClone) FetchStore(rel_path string) error {

	c.Count += 1

	remote_abspath := c.Source + rel_path
	local_abspath := path.Join(c.Dest, rel_path)

	_, err := os.Stat(local_abspath)

	if !os.IsNotExist(err) {

		change, _ := c.HasChanged(local_abspath, remote_abspath)

		if !change {
			return nil
		}

	} else {

		local_root := path.Dir(local_abspath)

		_, err := os.Stat(local_root)

		if os.IsNotExist(err) {
			log.Printf("create %s\n", local_root)
			os.MkdirAll(local_root, 0755)
		}
	}

	log.Printf("fetch '%s' and store in %s\n", remote_abspath, local_abspath)

	rsp, fetch_err := c.Fetch("GET", remote_abspath)

	if fetch_err != nil {
		log.Println(fetch_err)
		return fetch_err
	}

	contents, read_err := ioutil.ReadAll(rsp.Body)

	if read_err != nil {
		log.Println(read_err)
	}

	write_err := ioutil.WriteFile(local_abspath, contents, 0644)

	if write_err != nil {
		log.Println(write_err)
	}

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

	etag := rsp.Header.Get("Etag")
	remote_hash := strings.Replace(etag, "\"", "", -1)

	if local_hash == remote_hash {
		change = false
	}

	log.Printf("hash %s etag %s change %t\n", local_hash, remote_hash, change)

	return change, nil
}

func (c *WOFClone) Fetch(method string, url string) (*http.Response, error) {

	req, _ := http.NewRequest(method, url, nil)
	req.Close = true

     c.Connections += 1

	rsp, err := c.Client.Do(req)

	c.Connections -= 1

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
