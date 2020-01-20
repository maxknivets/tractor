.PHONY: build setup clobber dev

build: clobber dev/bin/tractor-agent dev/bin/tractor 

setup: dev/workspace dev/bin studio/node_modules
	make build

dev:
	./dev/bin/tractor-agent --dev

clobber:
	rm -rf dev/bin/tractor 

dev/bin:
	mkdir -p dev/bin

dev/bin/tractor-agent: dev/bin
	go build -o ./dev/bin/tractor-agent ./cmd/tractor-agent

dev/bin/tractor: dev/bin
	go build -o ./dev/bin/tractor ./cmd/tractor

dev/workspace:
	mkdir -p dev
	cp -r data/workspace dev/workspace
	mv dev/workspace/workspace.go.data dev/workspace/workspace.go

studio/node_modules:
	cd studio && yarn install
	cd studio && yarn link qmux qrpc

studio/shell/src-gen:
	cd studio/shell && yarn compile

