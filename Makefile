EXECUTABLE=rpmostree-exporter
PLATFORMS := linux/amd64 linux/arm64
LDFLAGS=-ldflags "-s -w"

.PHONY: all $(PLATFORMS) clean

all: $(PLATFORMS)

GO ?= go

$(PLATFORMS):
	CGO_ENABLED=0 GOOS=$(word 1, $(subst /, ,$@)) GOARCH=$(word 2, $(subst /, ,$@)) go build $(LDFLAGS) -o bin/$(EXECUTABLE)_$(word 1, $(subst /, ,$@))_$(word 2, $(subst /, ,$@))

clean:
	rm -rf bin/*