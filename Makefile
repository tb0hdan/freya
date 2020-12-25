BUILD = $(shell git rev-parse HEAD)
BDATE = $(shell date -u '+%Y-%m-%d_%I:%M:%S%p_UTC')
GO_VERSION = $(shell go version|awk '{print $$3}')
VERSION = $(shell cat ./VERSION)

all: build

build:
	@docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t tb0hdan/freya .

freya:
	@go build -a -trimpath -tags netgo -installsuffix netgo -v -x -ldflags "-s -w -X main.XZChecksum=$(shell sha256sum /usr/bin/xz |awk '{print $$1}')  -X main.MassDNSChecksum=$(shell sha256sum /massdns/bin/massdns |awk '{print $$1}') -X main.Build=$(BUILD) -X main.BuildDate=$(BDATE) -X main.GoVersion=$(GO_VERSION) -X main.Version=$(VERSION)" -o /freya *.go
	@strip -S -x /freya

docker-run:
	@docker run --env FREYA=$$FREYA --rm -it tb0hdan/freya

tag:
	@git tag -a v$(VERSION) -m v$(VERSION)
	@git push --tags

dockertag:
	@docker tag tb0hdan/freya tb0hdan/freya:v$(VERSION)
	@docker tag tb0hdan/freya tb0hdan/freya:latest
	@docker push tb0hdan/freya:v$(VERSION)
	@docker push tb0hdan/freya:latest
