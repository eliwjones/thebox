package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
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

	pipe := make(chan []structs.Option, len(symbols))
	for _, symbol := range symbols {
		fmt.Printf("Getting: %s\n", symbol)
		go getOptions(tda, symbol, pipe)
	}

	holder := make([]structs.Option, 0)
	for _, _ = range symbols {
		options := <-pipe
		if len(options) == 0 {
			fmt.Println("Received empty options slice.")
			continue
		}
		fmt.Printf("Received: %s\n", options[0].Underlying)
		holder = append(holder, options...)
	}

	now := time.Now()
	limit := now.AddDate(0, 0, 22).Format("20060102")
	fmt.Printf("LOCAL: %d:%d\n", now.Hour(), now.Minute())
	fmt.Printf("UTC:   %d:%d\n", now.UTC().Hour(), now.UTC().Minute())

	for _, option := range holder {
		if option.Expiration > limit {
			continue
		}
		stuff, err := json.Marshal(option)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		key := fmt.Sprintf("%s_%s_%s_%d", option.Underlying, option.Expiration, option.Type, option.Strike)
		lazyWriteFile("data", key, stuff)
	}
}

func getOptions(tda *tdameritrade.TDAmeritrade, symbol string, pipe chan []structs.Option) {
	options, _, err := tda.GetOptions(symbol)
	if err != nil {
		fmt.Printf("Got err: %s\n", err)
		return
	}
	// Write stock info here? or make frankenstein pipe to send both down?
	// Make new pipe just for stock?
	pipe <- options
}

func lazyWriteFile(folderName string, fileName string, data []byte) error {
	err := ioutil.WriteFile(folderName+"/"+fileName, data, 0777)
	if err != nil {
		os.MkdirAll(folderName, 0777)
		err = ioutil.WriteFile(folderName+"/"+fileName, data, 0777)
	}
	if err != nil {
		fmt.Printf("[LazyWriteFile] Could not WriteFile: %s\nErr: %s\n", folderName+"/"+fileName, err)
	}
	return err
}
