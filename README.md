# Tractor Prototype

Programmable computing environment

### Prerequisites
 * golang 1.13+
 * git 2.11+, make
 * node 10.11+,yarn
 * typescript `yarn global add typescript`

### Setup
Most of setup should be automated, but you need to also clone and link qtalk.
```
$ git clone https://github.com/manifold/qtalk
$ cd qtalk
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

