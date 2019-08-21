GOOS ?= $(shell go env GOOS)
ORG := github.com
OWNER := cloud-native-taiwan
REPOPATH ?= $(ORG)/$(OWNER)/labels-syncker

$(shell mkdir -p ./out)

.PHONY: build
build: out/labels-syncker

.PHONY: out/labels-syncker
out/labels-syncker:
	GOOS=$(GOOS) go build -ldflags="-s -w" -a -o $@ main.go

.PHONY: clean
clean:
	rm -rf out/