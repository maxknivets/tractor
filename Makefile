.PHONY: build setup clobber dev

setup: dev/workspace dev/bin extension/node_modules
	make build
	
build: clobber dev/bin/tractor-agent dev/bin/tractor extension/out

clobber:
	rm -rf dev/bin/tractor 
	rm -rf extension/out

dev/bin/tractor-agent: dev/bin
	go build -o ./dev/bin/tractor-agent ./cmd/tractor-agent

dev/bin/tractor: dev/bin
	go build -o ./dev/bin/tractor ./cmd/tractor

dev: dev/bin/tractor
	./dev/bin/tractor dev

extension/node_modules:
	cd extension && yarn link qmux qrpc
	cd extension && yarn install

extension/out: extension/node_modules
	cd extension && yarn compile

dev/workspace:
	mkdir -p dev
	cp -r data/workspace dev/workspace
	mv dev/workspace/workspace.go.data dev/workspace/workspace.go

dev/bin:
	mkdir -p dev/bin