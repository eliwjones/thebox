Quick Start
===========

1. If you don't have GO, go get it [http://golang.org/doc/install]

2. Clone and test The Box. 
```
$ mkdir -p $GOPATH/src/github.com/eliwjones
$ cd $GOPATH/src/github.com/eliwjones
$ git clone git@github.com:eliwjones/thebox.git
$ cd thebox
$ for i in `ls -d */`; do go test -v $i/*.go; done;
=== RUN Test_Destiny
--- PASS: Test_Destiny (0.00 seconds)
PASS
ok  	command-line-arguments	0.003s
=== RUN Test_Dispatcher
--- PASS: Test_Dispatcher (0.00 seconds)
PASS
ok  	command-line-arguments	0.002s
=== RUN Test_Money
--- PASS: Test_Money (0.00 seconds)
PASS
ok  	command-line-arguments	0.002s
?   	command-line-arguments	[no test files]
```
