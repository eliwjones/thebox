package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util/funcs"

	"flag"
	"fmt"
	"os"
)

var (
	symbols  = []string{}
	root_dir = flag.String("root_dir", "", "Where to find config file, 'log' and 'data' directories?")
)

func init() {
	flag.Parse()

	if *root_dir == "" {
		fmt.Printf("Please specify -root_dir.")
		os.Exit(1)
	}
}

func main() {
	lines, _ := funcs.GetConfig(*root_dir + "/config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]
	jsess := lines[3]
	for _, symbol := range lines[4:] {
		if symbol == "" {
			continue
		}
		symbols = append(symbols, symbol)
	}

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		lines[3] = tda.JsessionID
		funcs.UpdateConfig(*root_dir+"/config", lines)
	}

	var c = collector.New(*root_dir, tda)

	pipe := make(chan bool, len(symbols))
	for _, symbol := range symbols {
		fmt.Printf("Getting: %s\n", symbol)
		go c.Collect(symbol, pipe)
	}

	for _, _ = range symbols {
		result := <-pipe
		if !result {
			fmt.Println("Received err result.")
			continue
		}
	}
}
