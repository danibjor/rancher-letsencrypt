# These env vars have to be set in the CI
# GITHUB_TOKEN
# DOCKER_HUB_TOKEN

.PHONY: build deps test release clean push image ci-compile build-dir ci-dist dist-dir ci-release version help

GO111MODULE=on
CGO_ENABLED=0

DIST_OS=$(if $(GOOS),$(GOOS),$(shell go env GOOS))
DIST_ARCH=$(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
LINTER=$(shell go env GOPATH)

PROJECT := rancher-letsencrypt
BIN_FILE_NAME := $(PROJECT)
BIN_OUTPUT := dist/$(BIN_FILE_NAME)

SOURCE_FILES = $(shell git ls-files '*.go' | grep -v '^vendor/')

DOCKER_IMAGE := smujaddid/$(PROJECT)
VERSION := $(shell cat VERSION)
SHA := $(shell git rev-parse --short HEAD)

default: clean checks build

all: help

help:
	@echo "make build - build binary in the current environment"
	@echo "make deps - install build dependencies"
	@echo "make vet - run vet & gofmt checks"
	@echo "make test - run tests"
	@echo "make clean - Duh!"
	@echo "make checks - run golangci-lint"
	@echo "make version - show app version"

clean:
	@echo "Cleaning build files"
	go clean
	rm -rf dist/ cover.out

build: clean version
	go build -ldflags "-X main.Version=$(VERSION) -X main.Git=$(SHA)" -o $(BIN_OUTPUT)

version:
	@echo $(VERSION) $(SHA)

vet:
	scripts/vet

test:
	go test -v -cover ./...

checks:
	$(shell go env GOPATH)/bin/golangci-lint run

fmt:
	gofmt -s -l -w $(SOURCE_FILES)

docker-local: build
	docker build -t $(DOCKER_IMAGE):test -f ./dockerfiles/Dockerfile.local .

docker-local-run:
	docker run -it --rm -d --name test --env-file .env $(DOCKER_IMAGE)

docker-local-run-bash:
	docker run -it --rm -d --name test --env-file .env $(DOCKER_IMAGE) /bin/bash

docker-local-stop:
	docker stop test

docker-local-shell:
	docker exec -it test /bin/bash

docker-dev:
	docker build -t $(DOCKER_IMAGE):dev-$(SHA) -f ./dockerfiles/Dockerfile.dev .
