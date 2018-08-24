.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO = go
BINARY_DIR=bin

GODIRS_NOVENDOR = $(shell go list ./... | grep -v /vendor/)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PACKAGE_BUILD_INFO=github.com/avarabyeu/rpquiz

BUILD_INFO_LDFLAGS=-ldflags "-extldflags '"-static"' -X ${PACKAGE_BUILD_INFO}/util.buildDate=${BUILD_DATE} -X ${PACKAGE_BUILD_INFO}/util.version=${version}"

.PHONY: vendor run

help:
	@echo "build		        - go build f"
	@echo "test       			- go test"
	@echo "checkstyle 			- gofmt+golint+misspell"

vendor:
	dep ensure

test:
	$(GO) test ${GODIRS_NOVENDOR}

checkstyle:
#	gometalinter --vendor ./... --fast --disable=gas --disable=gosec --disable=gotype --deadline 10m

checkstyle-deep:
	gometalinter --vendor ./... --fast --disable=gas --disable=gosec --deadline 10m

fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}

build: checkstyle test
	CGO_ENABLED=0 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${BINARY_DIR}/rpquiz ./


run:
	realize start

clean:
	if [ -d ${BINARY_DIR} ] ; then rm -r ${BINARY_DIR} ; fi
	if [ -d 'build' ] ; then rm -r 'build' ; fi
