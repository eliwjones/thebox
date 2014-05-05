Quick Start
===========

1. If you don't have GO, go get it [http://golang.org/doc/install]

2. Clone and test The Box. 
```
$ mkdir -p $GOPATH/src/github.com/eliwjones
$ cd $GOPATH/src/github.com/eliwjones
$ git clone git@github.com:eliwjones/thebox.git
$ cd thebox
$ go test -v money/*
=== RUN Test_Money
--- PASS: Test_Money (0.00 seconds)
PASS
$ go test -v destiny/*
=== RUN Test_Destiny
--- PASS: Test_Destiny (0.00 seconds)
PASS
```
