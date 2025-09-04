# Inspired by https://github.com/gofiber/fiber/blob/7b3a36f22fc1166ceb9cb78cf69b3a2f95d077da/Makefile
.PHONY: help all align audit build clean format lint run tidy up

version=$(shell cat VERSION 2>/dev/null)

help:
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## all: 🚀 Run pre-commit tasks
all: audit align tidy format lint build

## align: 📏 Optimize struct fields
align: ## https://github.com/dkorunic/betteralign/releases
	go run github.com/dkorunic/betteralign/cmd/betteralign@v0.7.3 -apply ./...

## audit: 🚀 Conduct quality checks
audit: ## https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
	go mod verify
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 -show verbose ./...

build: clean
	CGO_ENABLED=0 \
		go build -trimpath -buildmode=pie \
		-ldflags="-s -w -X main.version=${version}" \
		-o bin/pinnacle-${version}.bin .

## clean: 🧹 Remove artifacts
clean:
	rm -rf ./bin

## format: 🎨 Fix code formatting
format: ## https://github.com/mvdan/gofumpt/releases
	go run mvdan.cc/gofumpt@v0.9.0 -w -l .

## lint: 🚨 Run lint checks
lint: ## https://github.com/golangci/golangci-lint/releases
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0 run ./...

## run: ⚙️ Build and run app
run: build
	./bin/pinnacle-${version}.bin

## tidy: 📌 Clean dependencies
tidy:
	go mod tidy -v

## up: 🔺 Update dependencies
up:
	go get -u ./...
