package main

import (
	"flag"
	clone "github.com/whosonfirst/go-whosonfirst-clone"
	log "github.com/whosonfirst/go-whosonfirst-log"
	"io"
	"os"
	"runtime"
	"time"
)

func main() {

	var source = flag.String("source", "https://whosonfirst.mapzen.com/data/", "Where to look for files")
	var dest = flag.String("dest", "", "Where to write files")
	var procs = flag.Int("procs", (runtime.NumCPU() * 2), "The number of concurrent processes to clone data with")
	var loglevel = flag.String("loglevel", "info", "The level of detail for logging")
	var skip_existing = flag.Bool("skip-existing", false, "Skip existing files on disk (without checking for remote changes)")
	var strict = flag.Bool("strict", false, "Exit (1) if any meta file fails cloning")

	flag.Parse()
	args := flag.Args()

	writer := io.MultiWriter(os.Stdout)

	logger := log.NewWOFLogger("[wof-clone-metafiles] ")
	logger.AddLogger(writer, *loglevel)

	cl := clone.NewWOFClone(*source, *dest, *procs, logger)

	start := time.Now()

	for _, file := range args {

		err := cl.CloneMetaFile(file, *skip_existing)

		if err != nil {
			logger.Error("failed to clone %s, because %v", file, err)

			if *strict {
				os.Exit(1)
			}
		}
	}

	since := time.Since(start)
	secs := float64(since) / 1e9

	logger.Info("processed %d files (ok: %d error: %d skipped: %d) in %f seconds\n", cl.Scheduled, cl.Success, cl.Error, cl.Skipped, secs)
	os.Exit(0)
}
