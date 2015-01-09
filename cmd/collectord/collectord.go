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
	id       = flag.String("id", "", "In case one is multiple actions with same root_dir.")
	action   = flag.String("action", "", "'clean', 'collect', 'migrate' or 'process_stream'?")
	reckless = flag.Bool("reckless", false, "Request and save data ignoring trading time and day ranges.")
	root_dir = flag.String("root_dir", "", "Where to find config file, 'log' and 'data' directories?")
	start    = flag.String("start", "", "Starting Timestamp")
	end      = flag.String("end", "", "Ending Timestamp")
	yymmdd   = flag.String("yymmdd", "", "For '-action clean' need <YYMMDD> to clean.")
)

func init() {
	flag.Parse()

	if *root_dir == "" {
		fmt.Printf("Please specify -root_dir.\n")
		os.Exit(1)
	}
	if *action == "" {
		fmt.Printf("Please specify -action. ('collect', 'process_stream', 'clean' or 'migrate')\n")
		os.Exit(1)
	}
	if (*action == "clean" || *action == "migrate") && *yymmdd == "" {
		fmt.Printf("If performing '%s' action, must specify -yymmdd.\n", *action)
		os.Exit(1)
	}
	if *action == "process_stream" && (*start == "" || *end == "") {
		fmt.Printf("'process_stream' requires -start, and -end.\n")
		os.Exit(1)
	}
}

func main() {
	c := collector.New(*action+*id, *root_dir)
	c.Reckless = *reckless

	switch *action {
	case "collect":
		collect(c)
	case "process_stream":
		c.ProcessStream(*start, *end)
	case "clean":
		collector.Clean(*root_dir, *yymmdd)
	case "migrate":
		err := collector.Migrate(*root_dir, *yymmdd)
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

	for _, symbol := range lines[4:] {
		if symbol == "" {
			continue
		}
		err := c.Collect(symbol)
		if err != nil {
			fmt.Println(err)
		}
	}

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		lines[3] = tda.JsessionID
		funcs.UpdateConfig(*root_dir+"/config", lines)
	}

	c.Adapter = tda

	c.RunOnce()
}
