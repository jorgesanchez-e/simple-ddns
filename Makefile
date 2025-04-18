APPNAME ?= simple-ddns

# used by `test` target
export REPORTS_DIR=./reports
# used by lint target
export GOLANGCILINT_VERSION=v1.54.2

build: clean
	mkdir -p build
	GOOS=$(GOOS) GOARCH=$(GOARCH) APPNAME=$(APPNAME) ./scripts/build

run: build
	./build/${APPNAME}

test:
	./scripts/unit-test

test-report:
	./scripts/show-tests

lint:
	./scripts/lint

clean:
	APPNAME=$(APPNAME) ./scripts/clean

docker:
	APPNAME=$(APPNAME) DOCKER_TAG=$(DOCKER_TAG) ./scripts/docker

.PHONY: build run test test-report lint clean docker
