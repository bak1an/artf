BINARY=artf
PKG=github.com/bak1an/artf

BUILD := $(shell date +%FT%T%z)
GIT_REV := $(shell git rev-parse HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "untagged")

LDFLAGS=-ldflags "-s -w -X ${PKG}/version.build=${BUILD} -X ${PKG}/version.gitRev=${GIT_REV} -X ${PKG}/version.gitBranch=${GIT_BRANCH} -X ${PKG}/version.gitTag=${GIT_TAG}"
BUILDFLAGS=-trimpath

.DEFAULT_GOAL := build

.PHONY: fmt test vet nils check clean local-build linux-amd64-build linux-arm64-build dist-gzip dist-sha256 dist build integration

fmt:
	go fmt ./... && go tool goimports -w .

test:
	go test -v -race ./...

vet:
	go vet -v ./...

nils:
	go tool nilaway -test=false ./...

integration:
	./tests/integration_test.py

check: vet nils test integration

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -d ./dist ] ; then rm -rf ./dist ; fi

local-build:
	go build ${BUILDFLAGS} -v ${LDFLAGS} -o ${BINARY}

linux-amd64-build:
	mkdir -p ./dist && GOOS=linux GOARCH=amd64 go build -v ${BUILDFLAGS} ${LDFLAGS} -o ./dist/artf-linux-amd64

linux-arm64-build:
	mkdir -p ./dist && GOOS=linux GOARCH=arm64 go build -v ${BUILDFLAGS} ${LDFLAGS} -o ./dist/artf-linux-arm64

dist-gzip:
	cd ./dist && tar --no-xattrs --disable-copyfile -czf artf-linux-amd64.tar.gz artf-linux-amd64
	cd ./dist && tar --no-xattrs --disable-copyfile -czf artf-linux-arm64.tar.gz artf-linux-arm64

dist-sha256:
	cd ./dist && sha256sum artf-linux-amd64.tar.gz > artf-linux-amd64.tar.gz.sha256
	cd ./dist && sha256sum artf-linux-arm64.tar.gz > artf-linux-arm64.tar.gz.sha256

dist: clean linux-amd64-build linux-arm64-build dist-gzip dist-sha256

build: clean local-build
