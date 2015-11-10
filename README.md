# go-whosonfirst-clone

Tools and libraries for cloning (not syncing) Who's on First data to your local machine.

This is still very much a work in progress so you might want to wait before using it. For the adventurous...

## Prereqs

Install GO! There are package installers for Mac and Windows, and build from source options.

* https://golang.org/dl/

TIP: On Mac, verify your bash profile includes:

    export PATH=$PATH:/usr/local/go/bin

## Installation

Clone the repo:

    git clone git@github.com:whosonfirst/go-whosonfirst-clone.git

Move into the repo's directory with:

    cd go-whosonfirst-clone

Setup that director as a new GO workspace:

    export GOPATH=`pwd`

Install a few WOF-Clone dependencies

    make deps

Which logs which dependencies are being installed:

```
go get -u "github.com/whosonfirst/go-whosonfirst-csv"
go get -u "github.com/whosonfirst/go-whosonfirst-log"
go get -u "github.com/whosonfirst/go-whosonfirst-pool"
go get -u "github.com/whosonfirst/go-whosonfirst-utils"
go get -u "github.com/jeffail/tunny"
```

Compile the wof-clone Go tools to binary:

```
make bin
```

## Usage

```
$> ./bin/wof-clone -h
Usage of ./bin/wof-clone:
  -dest string
    	Where to write files
  -loglevel string
    	    The level of detail for logging (default "info")
  -procs int
    	 The number of concurrent processes to clone data with (default 8)
  -skip-existing
	Skip existing files on disk (without checking for remote changes)
  -source string
    	  Where to look for files (default "https://whosonfirst.mapzen.com/data/")
```

### Example

```
$>./bin/wof-clone -dest ../tmp/ -skip-existing /usr/local/mapzen/whosonfirst-data/meta/wof-microhood-latest.csv
[clone] 10:55:03.713219 [info] processed 35 files (ok: 0 error: 0 skipped: 35) in 0.000877 seconds
```

## See also

* https://github.com/whosonfirst/go-whosonfirst-howto
