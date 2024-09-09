version=$(shell cat VERSION 2>/dev/null)

.PHONY: all audit build clean lint run tidy

all: audit lint build

audit:
	go mod verify
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.3 -show verbose ./...

build: clean
	CGO_ENABLED=0 \
		go build -trimpath -buildmode=pie \
		-ldflags="-s -w -X main.version=${version}" \
		-o bin/pinnacle-${version}.bin .

clean:
	rm -rf ./bin

lint: tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3 run ./...

run: build
	./bin/pinnacle-${version}.bin

tidy:
	go mod tidy -v
	go run mvdan.cc/gofumpt@v0.7.0 -w -l .