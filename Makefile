prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-clone; then rm -rf src/github.com/whosonfirst/go-whosonfirst-clone; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-clone
	cp clone.go src/github.com/whosonfirst/go-whosonfirst-clone/

build:	deps fmt bin

deps:
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(shell pwd) go get -u "github.com/jeffail/tunny"

bin:	clone

clone:	fmt self
	@GOPATH=$(shell pwd) go build -o bin/wof-clone-metafiles cmd/wof-clone-metafiles.go

fmt:
	go fmt *.go
	go fmt cmd/*.go
