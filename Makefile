#SHELL=/usr/bin/env bash

CLEAN:=
BINS:=
DATE_TIME=`date +'%Y%m%d %H:%M:%S'`
COMMIT_ID=`git rev-parse --short HEAD`
DOCKER := $(shell which docker)

build: tidy
	go build -ldflags "-s -w -X 'main.BuildTime=${DATE_TIME}' -X 'main.GitCommit=${COMMIT_ID}'"
.PHONY: build
BINS+=portmap
.PHONY+=portmap

install: tidy
	go install -ldflags "-s -w -X 'main.BuildTime=${DATE_TIME}' -X 'main.GitCommit=${COMMIT_ID}'"
.PHONY+=install

tidy:
	go mod tidy
.PHONY+=tidy

clean:
	rm -rf $(BINS) $(CLEAN)

