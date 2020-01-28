# Tractor Prototype

These are instructions for development, which is the only way to use Tractor right now.

### Prerequisites
 * golang 1.13+
 * git 2.11+
 * node 10.x.x (not 12.x.x)
 * python (for node-gyp)
 * make
 * yarn `npm i -g yarn`
 * typescript `yarn global add typescript`

 * for linux: 
   * gtk+3.0-dev
   * webkit2gtk-dev
   * libappindicator-dev
   * g++
 * for mac: 
   * XCode Command Line Tools

See what versions you have with `make versions`:
```
$ make versions
go version go1.13.3 darwin/amd64
node v10.16.1
git version 2.24.1
yarn 1.21.1
typescript Version 3.0.3
```
Node is a tricky one because it must be less than version 12, which 
apparently means the 10.x line. On OS X installing with `brew install node@10`
may still result in version 12. You may need to force `brew link` the older version.

### Setup
Just clone and setup tractor:
```
$ git clone https://github.com/manifold/tractor
$ cd tractor
$ make setup
```

### Running / Development
Run the agent in development mode with:
```
$ make dev
```
Tractor Studio will be running on `http://localhost:3000` but you want to open it with a 
path to a Tractor workspace. A development workspace is created for you at `local/workspace`
which should show up in the Tractor systray menu as `dev`. Clicking `dev` in the menu will
launch Studio in your browser opened to that workspace.
