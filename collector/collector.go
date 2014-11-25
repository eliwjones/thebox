package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
	"os"
	"time"
)

var root_dir = "collector"

func Collect(tda *tdameritrade.TDAmeritrade, symbol string, pipe chan bool) {
	now := time.Now().UTC()
	filename := now.Format("20060102")
	timestamp := now.Format("150405")
	logpath := fmt.Sprintf("%s/log", root_dir)
	
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		message := "No need for Sat, Sun."
		lazyAppendFile(logpath, filename, timestamp + " : " + message)
		fmt.Println(message)
		pipe <- false
		return
	}
	early := "13:28"
	late := "21:02"
	tooEarly, _ := time.Parse("20060102 15:04", now.Format("20060102")+" "+early)
	tooLate, _ := time.Parse("20060102 15:04", now.Format("20060102")+" "+late)
	// Hamfisted block before 13:30 UTC and after 21:00 UTC.
	if now.Before(tooEarly) || now.After(tooLate) {
		message := fmt.Sprintf("Time %s is before %s UTC or after %s UTC", now.Format("15:04:05"), early, late)
		lazyAppendFile(logpath, filename, timestamp + " : " + message)
		fmt.Println(message)
		pipe <- false
		return
	}

	limit := now.AddDate(0, 0, 22).Format("20060102")

	options, stock, err := tda.GetOptions(symbol)
	if err != nil {
		message := fmt.Sprintf("Got err: %s", err)
		lazyAppendFile(logpath, filename, timestamp + " : " + message)
		fmt.Println(message)
		pipe <- false
		return
	}

	es, _ := funcs.Encode(&stock, funcs.StockEncodingOrder)

	path := fmt.Sprintf("%s/data/%s/s", root_dir, stock.Symbol)
	lazyAppendFile(path, filename, timestamp+","+es)

	for _, option := range options {
		if option.Expiration > limit {
			continue
		}
		eo, err := funcs.Encode(&option, funcs.OptionEncodingOrder)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		path := fmt.Sprintf("%s/data/%s/o/%s/%s", root_dir, option.Underlying, option.Expiration, option.Type)
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
