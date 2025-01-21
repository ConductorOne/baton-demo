GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
BUILD_DIR = dist/${GOOS}_${GOARCH}

ifeq ($(GOOS),windows)
OUTPUT_PATH = ${BUILD_DIR}/baton-demo.exe
LAMBDA_CLIENT_OUTPUT_PATH = ${BUILD_DIR}/baton-lambda-client.exe
LAMBDA_SERVER_OUTPUT_PATH = ${BUILD_DIR}/baton-lambda-server.exe

else
OUTPUT_PATH = ${BUILD_DIR}/baton-demo
LAMBDA_CLIENT_OUTPUT_PATH = ${BUILD_DIR}/baton-lambda-client
LAMBDA_SERVER_OUTPUT_PATH = ${BUILD_DIR}/baton-lambda-server
endif

.PHONY: build
build:
	go build -o ${OUTPUT_PATH} ./cmd/baton-demo
	go build -o ${LAMBDA_CLIENT_OUTPUT_PATH} ./cmd/baton-lambda-client
	go build -o ${LAMBDA_SERVER_OUTPUT_PATH} ./cmd/baton-lambda-server

.PHONY: build-lambda-server
build-lambda-server:
	go build -o ${LAMBDA_SERVER_OUTPUT_PATH} ./cmd/baton-lambda-server


.PHONY: update-deps
update-deps:
	go get -d -u ./...
	go mod tidy -v
	go mod vendor

.PHONY: add-deps
add-dep:
	go mod tidy -v
	go mod vendor

.PHONY: lint
lint:
	golangci-lint run

.PHONY: sam-build
sam-build:
	DOCKER_HOST=unix://$(HOME)/.docker/run/docker.sock sam build

.PHONY: sam-run
sam-run: sam-build
	DOCKER_HOST=unix://$(HOME)/.docker/run/docker.sock sam local start-lambda

