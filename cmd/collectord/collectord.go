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
	action   = flag.String("action", "", "'clean', 'collect' or 'migrate'?")
	root_dir = flag.String("root_dir", "", "Where to find config file, 'log' and 'data' directories?")
	yymmdd   = flag.String("yymmdd", "", "For '-action clean' need <YYMMDD> to clean.")
)

func init() {
	flag.Parse()

	if *root_dir == "" {
		fmt.Printf("Please specify -root_dir.\n")
		os.Exit(1)
	}
	if *action == "" {
		fmt.Printf("Please specify -action. ('clean' or 'collect')\n")
		os.Exit(1)
	}
	if (*action == "clean" || *action == "migrate") && *yymmdd == "" {
		fmt.Printf("If performing '%s' action, must specify -yymmdd.\n", *action)
		os.Exit(1)
	}
}

func main() {
	var c = collector.New(*root_dir)

	switch *action {
	case "collect":
		collect(c)
	case "clean":
		c.Clean(*yymmdd)
	case "migrate":
		err := c.Migrate(*yymmdd)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func collect(c *collector.Collector) {
	lines, _ := funcs.GetConfig(*root_dir + "/config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]
	jsess := lines[3]

	symbols := []string{}
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

	c.Adapter = tda

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
