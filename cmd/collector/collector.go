package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
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
	early := "13:28"
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

	filename := now.Format("20060102")
	timestamp := now.Format("150405")

	es, _ := funcs.Encode(&stock, funcs.StockEncodingOrder)

	path := fmt.Sprintf("data/%s/s", stock.Symbol)
	lazyAppendFile(path, filename, timestamp+","+es)

	for _, option := range options {
		if option.Expiration > limit {
			continue
		}
		eo, err := funcs.Encode(&option, funcs.OptionEncodingOrder)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		path := fmt.Sprintf("data/%s/o/%s/%s", option.Underlying, option.Expiration, option.Type)
		lazyAppendFile(path, filename, timestamp+","+eo)
	}

	pipe <- true
}

func lazyAppendFile(folderName string, fileName string, data string) error {
	f, err := os.OpenFile(folderName+"/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		os.MkdirAll(folderName, 0777)
		f, err = os.OpenFile(folderName+"/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	}
	if err != nil {
		fmt.Printf("[lazyAppendFile] Could not Open File: %s\nErr: %s\n", folderName+"/"+fileName, err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(data + "\n")
	if err != nil {
		fmt.Printf("[lazyAppendFile] Could not AppendFile: %s\nErr: %s\n", folderName+"/"+fileName, err)
		return err
	}

	return nil
}
