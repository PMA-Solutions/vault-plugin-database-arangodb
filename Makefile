
TOOL?=vault-plugin-database-arangodb
TEST?=$$(go list ./... | grep -v /vendor/ | grep -v teamcity)
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr
BUILD_TAGS?=${TOOL}
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
GO_TEST_CMD?=go test -v

GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

# dev starts up `vault` from your $PATH, then builds the couchbase
# plugin, registers it with vault and enables it.
# A ./tmp dir is created for configs and binaries, and cleaned up on exit.
dev: fmtcheck
	@CGO_ENABLED=0 BUILD_TAGS='$(BUILD_TAGS)' VAULT_DEV_BUILD=1 sh -c "'$(CURDIR)/scripts/build.sh'"


build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o ./build/plugins/arangodb-database-plugin ./cmd/arangodb-database-plugin/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./build/plugins/

enable:
	vault write database/config/arango \
    plugin_name=arangodb-database-plugin \
    allowed_roles="my-role" \
    username="root" \
    password="root" \
    connection_url="http://localhost:8529"

	vault write database/roles/my-role \
    db_name=arango \
		creation_statements='{"collection_grants": [{"db": "hive", "access": "rw"}]}' \
    default_ttl="1m" \
    max_ttl="24h"

clean:
	rm -f ./build/plugins/arangodb-database-plugin

fmt:
	go fmt $$(go list ./...)

# test runs the unit tests and vets the code
test: fmtcheck
	CGO_ENABLED=0 VAULT_TOKEN= ${GO_TEST_CMD} -tags='$(BUILD_TAGS)' $(TEST) $(TESTARGS) -count=1 -timeout=5m -parallel=4

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

.PHONY: dev build clean fmt start enable fmtcheck fmt
