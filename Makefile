VERSION := $(shell cat VERSION)
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

export VERSION
export COMMIT_HASH
export BUILD_TIME

all:
	@echo "Building with version: $(VERSION), commit: $(COMMIT_HASH), time: $(BUILD_TIME)"
	go build -a -trimpath -ldflags "-w -s -X 'github.com/dbnski/query-stats/Version=$(VERSION)' -X 'github.com/dbnski/query-stats/CommitHash=$(COMMIT_HASH)' -X 'github.com/dbnski/query-stats/Build=dev' -X 'github.com/dbnski/query-stats/BuildTime=$(BUILD_TIME)'" -o query-stats .
