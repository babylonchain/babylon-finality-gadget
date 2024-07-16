.PHONY: lint test mock-gen

CUR_DIR := $(shell pwd)
MOCKS_DIR=$(CUR_DIR)/testutil/mocks
MOCKGEN_REPO=github.com/golang/mock/mockgen
MOCKGEN_VERSION=latest
MOCKGEN_CMD=mockgen

mock-gen:
	# TODO: Install mockgen if not installed
	mockgen -source=sdk/client/expected_clients.go -package mocks -destination $(MOCKS_DIR)/expected_clients_mock.go
	mockgen -source=sdk/client/interface.go -package mocks -destination $(MOCKS_DIR)/sdkclient_mock.go

test:
	go test -race ./... -v

lint:
	golangci-lint run