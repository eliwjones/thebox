package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
)

var (
	symbols = [3]string{"AAPL", "GOOG", "BABA"}
)

func main() {
	id, pass, sid, jsess, _ := funcs.GetConfig()

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		funcs.UpdateConfig(id, pass, sid, tda.JsessionID)
	}

	pipe := make(chan bool, len(symbols))
	for _, symbol := range symbols {
		fmt.Printf("Getting: %s\n", symbol)
		go collector.Collect(tda, symbol, pipe)
	}

	for _, _ = range symbols {
		result := <-pipe
		if !result {
			fmt.Println("Received err result.")
			continue
		}
	}
}
