.PHONY: lint test mock-gen

CUR_DIR := $(shell pwd)
MOCKS_DIR=$(CUR_DIR)/testutil/mocks
MOCKGEN_REPO=github.com/golang/mock/mockgen
MOCKGEN_VERSION=latest
MOCKGEN_CMD=mockgen

mock-gen:
	# TODO: Install mockgen if not installed
	mockgen -source=sdk/bbnclient/interface.go -package mocks -destination $(MOCKS_DIR)/bbnclient_mock.go
	mockgen -source=sdk/btcclient/interface.go -package mocks -destination $(MOCKS_DIR)/btcclient_mock.go
	mockgen -source=sdk/cwclient/interface.go -package mocks -destination $(MOCKS_DIR)/cwclient_mock.go
	mockgen -source=sdk/client/interface.go -package mocks -destination $(MOCKS_DIR)/sdkclient_mock.go

test:
	go test -race ./... -v

lint:
	golangci-lint run