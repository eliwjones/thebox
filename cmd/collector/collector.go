package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"

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

	pipe := make(chan bool, len(symbols))
	for _, symbol := range symbols {
		fmt.Printf("Getting: %s\n", symbol)
		go getData(tda, symbol, pipe)
	}

	for _, _ = range symbols {
		result := <-pipe
		if !result {
			fmt.Println("Received err result.")
			continue
		}
	}
}

func getData(tda *tdameritrade.TDAmeritrade, symbol string, pipe chan bool) {
	now := time.Now().UTC()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		fmt.Println("No need for Sat, Sun.")
		pipe <- false
		return
	}
	early := "13:38"
	late := "21:02"
	tooEarly, _ := time.Parse("20060102 15:04", now.Format("20060102")+" "+early)
	tooLate, _ := time.Parse("20060102 15:04", now.Format("20060102")+" "+late)
	// Hamfisted block before 13:30 UTC and after 21:00 UTC.
	if now.Before(tooEarly) || now.After(tooLate) {
		fmt.Printf("Time %s is before %s UTC or after %s UTC\n", now.Format("15:04:05"), early, late)
		pipe <- false
		return
	}

	limit := now.AddDate(0, 0, 22).Format("20060102")

	options, stock, err := tda.GetOptions(symbol)
	if err != nil {
		fmt.Printf("Got err: %s\n", err)
		pipe <- false
		return
	}

	es, _ := funcs.Encode(&stock, funcs.StockEncodingOrder)
	key := fmt.Sprintf("%s", now.Format("20060102_150405"))
	path := fmt.Sprintf("data/%s/s", stock.Symbol)
	lazyWriteFile(path, key, []byte(es))

	for _, option := range options {
		if option.Expiration > limit {
			continue
		}
		eo, err := funcs.Encode(&option, funcs.OptionEncodingOrder)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		key := fmt.Sprintf("%s_%d", now.Format("20060102_150405"), option.Strike)
		path := fmt.Sprintf("data/%s/o/%s/%s", option.Underlying, option.Expiration, option.Type)
		lazyWriteFile(path, key, []byte(eo))
	}

	pipe <- true
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
