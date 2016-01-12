package main

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-clone"
	"github.com/whosonfirst/go-whosonfirst-log"
	"io"
	"os"
	"runtime"
)

func main() {

	/*
	   Important see the way 'source' is the fully qualified AWS/S3 path and not whosonfirst.mapzen.com/data?
	   That's a... thing. The pretty domain results in an endless array of HTTP errors that cascade in to too
	   many open filehandle errors and table-pounding. Because computers. (20160112/thisisaaronland)
	*/

	var source = flag.String("source", "https://s3.amazonaws.com/whosonfirst.mapzen.com/data/", "Where to look for files")
	var dest = flag.String("dest", "", "Where to write files")
	var procs = flag.Int("procs", (runtime.NumCPU() * 2), "The number of concurrent processes to clone data with")
	var loglevel = flag.String("loglevel", "info", "The level of detail for logging")
	var skip_existing = flag.Bool("skip-existing", false, "Skip existing files on disk (without checking for remote changes)")
	var force_updates = flag.Bool("force-updates", false, "Force updates to files on disk (without checking for remote changes)")
	var strict = flag.Bool("strict", false, "Exit (1) if any meta file fails cloning")

	flag.Parse()
	args := flag.Args()

	writer := io.MultiWriter(os.Stdout)

	logger := log.NewWOFLogger("[wof-clone-metafiles] ")
	logger.AddLogger(writer, *loglevel)

	cl, err := clone.NewWOFClone(*source, *dest, *procs, logger)

	if err != nil {
		logger.Error("failed to create new Clone instance, because %v", err)
		os.Exit(1)
	}

	for _, file := range args {

		err := cl.CloneMetaFile(file, *skip_existing, *force_updates)

		if err != nil {
			logger.Error("failed to clone %s, because %v", file, err)

			if *strict {
				os.Exit(1)
			}
		}
	}

	cl.Status()
	os.Exit(0)
}
