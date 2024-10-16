CGO_ENABLED ?= 0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOARM ?= $(shell go env GOARM)

BUILD_DIR ?= ./build
SVC = gophercon
DOCKER_IMAGE_NAME ?= ghcr.io/rodneyosodo/gophercon
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -ldflags "-s -w " -o ${BUILD_DIR}/$(SVC) cmd/main.go
endef

define make_docker
	docker build \
		--build-arg SVC=$(SVC) \
		--no-cache \
		--tag=$(DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile .
endef

define docker_push
	docker push $(DOCKER_IMAGE_NAME):$(VERSION)
	docker push $(DOCKER_IMAGE_NAME):latest
endef

.PHONY: build
build:
	@mkdir -p ${BUILD_DIR}
	$(call compile_service)

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

.PHONY: docker
docker: build
	$(call make_docker)

.PHONY: docker-push
docker-push: docker
	$(call docker_push)

.PHONY: run-binary
run-binary:
	@go run cmd/main.go

.PHONY: proto
proto:
	@protoc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative calculator/calculator.proto

.PHONY: lint
lint:
	@golangci-lint run  --config .golangci.yaml

.PHONY: all
all: build docker

.PHONY: help
help:
	@echo "Makefile for gophercon"
	@echo "Usage:"
	@echo "  make build - Build the binary"
	@echo "  make docker - Build the docker image"
	@echo "  make docker-push - Push the docker image"
	@echo "  make run-binary - Run the binary"
	@echo "  make proto - Generate protobuf files"
	@echo "  make lint - Lint the code"
	@echo "  make all - Build the binary and docker image"
	@echo "  make clean - Clean the build directory"
