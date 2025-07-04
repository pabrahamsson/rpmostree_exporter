EXECUTABLE=rpmostree_exporter
PLATFORMS := linux/amd64 linux/arm64
LDFLAGS=-ldflags "-s -w \
	-X github.com/prometheus/common/version.Version=${VERSION} \
	-X github.com/prometheus/common/version.BuildDate=`date +%FT%T%z` \
	-X github.com/prometheus/common/version.BuildUser=`whoami` \
	-X github.com/prometheus/common/version.Branch=`git branch --show-current`"

.PHONY: all $(PLATFORMS) clean

all: $(PLATFORMS)

GO ?= go

$(PLATFORMS):
	CGO_ENABLED=0 GOOS=$(word 1, $(subst /, ,$@)) GOARCH=$(word 2, $(subst /, ,$@)) go build $(LDFLAGS) -o bin/$(EXECUTABLE)_$(word 1, $(subst /, ,$@))_$(word 2, $(subst /, ,$@))

gotest:
	go test .

checksums:
	cd bin;sha256sum $(EXECUTABLE)* > $(EXECUTABLE)_checksums.txt;cd ..

clean:
	rm -rf bin/*
