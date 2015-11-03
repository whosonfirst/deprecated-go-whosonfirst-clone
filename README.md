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
go get -u "github.com/jeffail/tunny"
```

Compile the WOF-Clone GO tools to binary:

    make bin

## Run it

Then run it using:

    ./bin/wof-clone -dest /usr/local/mapzen/whosonfirst-data/data/ ../whosonfirst-data/meta/wof-*-latest.csv

## See also

* https://github.com/whosonfirst/go-whosonfirst-howto
