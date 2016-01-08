# go-whosonfirst-clone

Tools and libraries for cloning (not syncing) Who's on First data to your local machine.

This is still very much a work in progress so you might want to wait before using it. For the adventurous...

## Setup

### Go

Install [Go](https://golang.org). There are package installers for Mac and Windows, and build from source options.

* https://golang.org/dl/

TIP: On Mac, verify your bash profile includes:

    export PATH=$PATH:/usr/local/go/bin

Next clone this repo (assuming you haven't already done that :-)

```
$> git clone git@github.com:whosonfirst/go-whosonfirst-clone.git
```

## Installation

The easiest way to install all the dependencies and compile all of the code and command line tools is to use the handy Makefile that is included with this repository, like this:

```
$> cd go-whosonfirst-clone
$> make deps
$> make bin
```

In addition to fetching all the necessary dependencies this will clone the `go-whosonfirst-clone` packages in to the `src` directory (along with all the dependencies) which is a thing you need to do because of the way Go expects code to organized. It's kind of weird and annoying but also shouting-at-the-sky territory so the Makefile is designed to hide the bother from you.

If you don't have `make` installed on your computer or just want to do things by hand then [you should spend some time reading the Makefile](Makefile) itself. The revelant "targets" (which are the equivalent of commands in Makefile-speak) that you will need are `deps` for fetching dependencies, `self` for cloning files and `bin` for building the command line tools.

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
