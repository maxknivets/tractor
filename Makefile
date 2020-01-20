.PHONY: build setup clobber dev versions

build: clobber local/bin/tractor-agent local/bin/tractor 

setup: local/workspace local/bin studio/node_modules
	make build

dev:
	./local/bin/tractor-agent --dev

clobber:
	rm -rf local/bin/tractor 
	rm -rf local/bin/tractor-agent

versions:
	go version
	node --version
	git --version
	

local/bin:
	mkdir -p local/bin

local/bin/tractor-agent: local/bin
	go build -o ./local/bin/tractor-agent ./cmd/tractor-agent

local/bin/tractor: local/bin
	go build -o ./local/bin/tractor ./cmd/tractor

local/workspace:
	mkdir -p local
	cp -r data/workspace local/workspace
	mv local/workspace/workspace.go.data local/workspace/workspace.go

studio/node_modules:
	cd studio && yarn install
	cd studio && yarn link qmux qrpc

studio/extension/lib:
	cd studio/extension && yarn build

studio/shell/src-gen: studio/extension/lib
	cd studio/shell && yarn build

