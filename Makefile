.PHONY: totp

SOURCES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GIT_HASH = "NONE" #$(shell git rev-parse --short HEAD)
GO_DEF_FLAGS = -ldflags "-s -w -extldflags=-static -X main.githash=$(GIT_HASH)"

totp: $(SOURCES)
	@mkdir -p ./bin
	@go build $(GO_DEF_FLAGS) -o ./bin/totp-cli ./cmd
