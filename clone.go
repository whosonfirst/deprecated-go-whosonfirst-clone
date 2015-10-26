package main

import (
	"flag"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

var count int

var source = flag.String("source", "http://whosonfirst.mapzen.com/data/", "Where to look for files")
var dest = flag.String("dest", "", "Where to write files")

// PLEASE FOR TO MAKE ALL OF THIS IN TO A PROPER PACKAGE

func ParseFile(file string) error {

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
			FetchStore(rel_path)
		}()
	}

	wg.Wait()

	return nil
}

func FetchStore(rel_path string) error {

	count += 1

	remote_abspath := *source + rel_path
	local_abspath := path.Join(*dest, rel_path)

	// has_changed := false

	_, err := os.Stat(local_abspath)

	if ! os.IsNotExist(err) {
	   // Check whether file has changed here	

	   log.Printf("%s already exists\n", local_abspath)
	   return nil
	} else {

	  local_root := path.Dir(local_abspath)

	  _, err := os.Stat(local_root)

	  if os.IsNotExist(err) {
		log.Printf("create %s\n", local_root)
		os.MkdirAll(local_root, 0755)
		}
	}

	log.Printf("fetch '%s' and store in %s\n", remote_abspath, local_abspath)

	rsp, fetch_err := http.Get(remote_abspath)

	if fetch_err != nil {
		log.Fatal(fetch_err)
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

func main() {

	flag.Parse()
	args := flag.Args()

	start := time.Now()
	count = 0

	wg := new(sync.WaitGroup)

	for _, file := range args {

		wg.Add(1)

		go func() {
			defer wg.Done()
			log.Println(file)
			ParseFile(file)
		}()
	}

	wg.Wait()

	since := time.Since(start)
	secs := float64(since) / 1e9

	log.Printf("processed %d files in %f seconds\n", count, secs)
}
