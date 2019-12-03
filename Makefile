
setup: dev/workspace dev/bin extension/node_modules
	make build
	
build: dev/bin/tractor

dev/bin/tractor: dev/bin
	go build -o ./dev/bin/tractor ./cmd/tractor

dev: dev/bin/tractor
	./dev/bin/tractor dev

extension/node_modules:
	cd extension && yarn link qmux qrpc
	cd extension && yarn install
	cd extension && yarn compile

dev/workspace:
	mkdir -p dev
	cp -r data/workspace dev/workspace

dev/bin:
	mkdir -p dev/bin