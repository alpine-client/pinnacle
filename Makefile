version=$(shell cat VERSION 2>/dev/null)

.PHONY: all audit build clean lint tidy run

all: audit lint build

audit:
	go mod verify
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

build: clean
	CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.version=${version}" -o bin/pinnacle-${version}.bin .

clean:
	rm -rf ./bin

lint: tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2 run ./...

run: build
	./bin/pinnacle-${version}.bin

tidy:
	go mod tidy -v
	go run mvdan.cc/gofumpt@latest -w -l .