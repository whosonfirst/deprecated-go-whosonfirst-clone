prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-clone; then rm -rf src/github.com/whosonfirst/go-whosonfirst-clone; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-clone
	cp clone.go src/github.com/whosonfirst/go-whosonfirst-clone/

deps:
	go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	go get -u "github.com/whosonfirst/go-whosonfirst-log"
	go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	go get -u "github.com/jeffail/tunny"

bin:	clone

clone:	fmt self
	go build -o bin/wof-clone cmd/wof-clone.go

fmt:
	go fmt *.go
	go fmt cmd/*.go
