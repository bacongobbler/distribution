# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)


# Used to populate version variable in main package.
VERSION=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)

# Allow turning off function inlining and variable registerization
ifeq (${DISABLE_OPTIMIZATION},true)
	GO_GCFLAGS=-gcflags "-N -l"
	VERSION:="$(VERSION)-noopt"
endif

GO_LDFLAGS=-ldflags "-X `go list ./version`.Version=$(VERSION)"

.PHONY: all build binaries clean dep-restore dep-save dep-validate fmt lint test test-full vet
.DEFAULT: all
all: fmt vet lint build test binaries

AUTHORS: .mailmap .git/HEAD
	 git log --format='%aN <%aE>' | sort -fu > $@

# This only needs to be generated by hand when cutting full releases.
version/version.go:
	./version/version.sh > $@

${PREFIX}/bin/registry: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ./cmd/registry

${PREFIX}/bin/registry-api-descriptor-template: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ./cmd/registry-api-descriptor-template

${PREFIX}/bin/dist: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ./cmd/dist

${PREFIX}/bin/registry: $(GOFILES)
	@echo "+ $@"
	@go build -tags "${DOCKER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/registry

${PREFIX}/bin/digest:  $(GOFILES)
	@echo "+ $@"
	@go build -tags "${DOCKER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/digest

${PREFIX}/bin/registry-api-descriptor-template: $(GOFILES)
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ${GO_GCFLAGS} ./cmd/registry-api-descriptor-template

docs/spec/api.md: docs/spec/api.md.tmpl ${PREFIX}/bin/registry-api-descriptor-template
	./bin/registry-api-descriptor-template $< > $@

vet:
	@echo "+ $@"
	@go vet ./...

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . | grep -v Godeps/_workspace/src/ | tee /dev/stderr)" || \
		echo "+ please format Go code with 'gofmt -s'"

lint:
	@echo "+ $@"
	@test -z "$$(golint ./... | grep -v Godeps/_workspace/src/ | tee /dev/stderr)"

build:
	@echo "+ $@"
	@go build -v ${GO_LDFLAGS} ./...

test:
	@echo "+ $@"
	@go test -test.short ./...

test-full:
	@echo "+ $@"
	@go test ./...

binaries: ${PREFIX}/bin/registry ${PREFIX}/bin/registry-api-descriptor-template ${PREFIX}/bin/dist
	@echo "+ $@"

clean:
	@echo "+ $@"
	@rm -rf "${PREFIX}/bin/registry" "${PREFIX}/bin/registry-api-descriptor-template"
