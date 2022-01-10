BIN=./bin

.PHONY: all build

default: all

all: build

build:
	mkdir -p $(BIN)
	gox -osarch 'linux/amd64' -output '$(BIN)/asdi_{{.OS}}_{{.Arch}}'
	# gox -osarch 'darwin/arm' -output '$(BIN)/asdi_{{.OS}}_{{.Arch}}' -- unsupported
	gox -osarch 'darwin/amd64' -output '$(BIN)/asdi_{{.OS}}_{{.Arch}}'
	gox -osarch 'windows/amd64' -output '$(BIN)/asdi_{{.OS}}_{{.Arch}}'
