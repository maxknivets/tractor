.PHONY: build setup clobber dev versions studio kill qtalk

build: clobber local/bin/tractor-agent local/bin/tractor 

setup: local/workspace local/bin studio qtalk
	make build

dev:
	./local/bin/tractor-agent --dev

kill:
	@killall node || true
	@killall tractor-agent || true

clobber:
	rm -rf local/bin/tractor 
	rm -rf local/bin/tractor-agent

versions:
	@go version
	@echo "node $(shell node --version)"
	@git --version
	@echo "yarn $(shell yarn --version)"
	@echo "typescript $(shell tsc --version)"
	
qtalk:
	git submodule update --init --recursive


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
	mkdir -p ~/.tractor/workspaces
	rm ~/.tractor/workspaces/dev || true
	ln -fs $(PWD)/local/workspace ~/.tractor/workspaces/dev

studio: studio/node_modules studio/extension/lib studio/shell/src-gen

studio/node_modules:
	cd studio && yarn install
	cd studio && yarn link qmux qrpc

studio/extension/lib:
	cd studio/extension && yarn build

studio/shell/src-gen: studio/extension/lib
	cd studio/shell && yarn build

