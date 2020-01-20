# Tractor Prototype

These are instructions for development, which is the only way to use Tractor right now.

### Prerequisites
 * golang 1.13+
 * git 2.11+
 * node 10.11+ (<12.x)
 * python (for node. right???)
 * make
 * yarn `npm i -g yarn`
 * typescript `yarn global add typescript`
 * for linux: gtk+3.0-dev webkit2gtk-dev libappindicator-dev g++
 * for mac: XCode Developer Tools, if you haven't

### Setup
You first need to clone and link qtalk:
```
$ git clone https://github.com/manifold/qtalk
$ make -C qtalk link
```
Now we can clone and setup tractor:
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

