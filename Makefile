BUILD = $(shell git rev-parse HEAD)
BDATE = $(shell date -u '+%Y-%m-%d_%I:%M:%S%p_UTC')
GO_VERSION = $(shell go version|awk '{print $$3}')
VERSION = $(shell cat ./VERSION)

all: build

build:
	@docker build -t tb0hdan/freya .

freya:
	@go build -a -trimpath -tags netgo -installsuffix netgo -v -x -ldflags "-s -w -X main.MassDNSChecksum=$(shell sha256sum /massdns/bin/massdns |awk '{print $$1}')" -o /freya *.go
	@strip -S -x /freya

docker-run:
	@docker run --env FREYA=$$FREYA --rm -it freya

tag:
	@git tag -a v$(VERSION) -m v$(VERSION)
	@git push --tags

dockertag:
	@docker tag tb0hdan/freya tb0hdan/freya:v$(VERSION)
	@docker push tb0hdan/freya:v$(VERSION)
