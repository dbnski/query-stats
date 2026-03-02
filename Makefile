VERSION     := $(shell cat VERSION)
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_TIME  := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -w -s
LDFLAGS += -X main.Version=$(VERSION)
LDFLAGS += -X main.CommitHash=$(COMMIT_HASH)
LDFLAGS += -X main.Build=dev
LDFLAGS += -X main.BuildTime=$(BUILD_TIME)

all:
	@echo "Building with version: $(VERSION), commit: $(COMMIT_HASH), time: $(BUILD_TIME)"
	go build -a -trimpath -ldflags "$(LDFLAGS)" -o query-stats .
