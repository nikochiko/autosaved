BIN="./bin"

.PHONY: all build deploy-local

default: all

all: build deploy-local

deploy-local:
	mv $(BIN)/asdi ~/go/bin/

build:
	mkdir -p $(BIN)
	go build -o $(BIN)/asdi
