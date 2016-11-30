CWD=$(shell pwd)
GOPATH := $(CWD)/vendor:$(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-clone; then rm -rf src/github.com/whosonfirst/go-whosonfirst-clone; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-clone
	cp clone.go src/github.com/whosonfirst/go-whosonfirst-clone/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	rmdeps bin

deps:
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(shell pwd) go get -u "github.com/jeffail/tunny"

bin:	self
	@GOPATH=$(GOPATH) go build -o bin/wof-clone-metafiles cmd/wof-clone-metafiles.go

vendor-deps: rmdeps deps
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src vendor/src
	find vendor -name '.git' -print -type d -exec rm -rf {} +

fmt:
	go fmt *.go
	go fmt cmd/*.go
