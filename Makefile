all: build

build:
	@docker build -t freya .

freya:
	@go build -a -trimpath -tags netgo -installsuffix netgo -v -x -ldflags "-s -w -X main.MassDNSChecksum=$(shell sha256sum /massdns/bin/massdns |awk '{print $$1}')" -o /freya *.go
	@strip -S -x /freya

docker-run:
	@docker run --env FREYA=$$FREYA --rm -it freya
