package util

import (
	"time"
)

type ContractType int

const (
	OPTION ContractType = iota
	STOCK
)

type Subscription struct {
	Id         string           // What is id of thing you are subscribing to.
	Whoami     string           // Who are you in case we need to delete.
	Subscriber chan interface{} // Where to send info.
}

type Signal struct {
	Payload interface{}
	Wait    chan bool
}

type Message struct {
	Data  interface{}      // Shall this be an interface?
	Reply chan interface{} // Reply if needed..
}

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }
