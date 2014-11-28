package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type collector struct {
	root_dir string
	adapter  *tdameritrade.TDAmeritrade
}

func New(root_dir string, adapter *tdameritrade.TDAmeritrade) *collector {
	return &collector{root_dir: root_dir, adapter: adapter}
}

func (c *collector) Cleanup(date string) []error {
	errors := []error{}

	data_dir := c.root_dir + "/data"
	d, _ := os.Open(data_dir)
	defer d.Close()
	symbols, _ := d.Readdirnames(-1)
	for _, symbol := range symbols {
		options_dir := data_dir + "/" + symbol + "/o"
		e, err := os.Open(options_dir)
		if err != nil {
			//fmt.Printf("No 'o' subdir found for: %s!\n", data_dir)
			errors = append(errors, err)
			continue
		}
		defer e.Close()
		expirations, _ := e.Readdirnames(-1)
		for _, expiration := range expirations {
			if len(expiration) != len("20140101") {
				fmt.Printf("Bad Length for: %s!\n", expiration)
				continue
			}
			exp_dir := options_dir + "/" + expiration
			for _, _type := range []string{"c", "p"} {
				cleanup_file := exp_dir + "/" + _type + "/" + date
				contents, err := ioutil.ReadFile(cleanup_file)
				if err != nil {
					//fmt.Printf("Could not find cleanup_file\n\t%s\n\t%s\n", cleanup_file, err)
					errors = append(errors, err)
					continue
				}
				cleanupOptionFile(cleanup_file, contents)
			}
		}
		stock_file := data_dir + "/" + symbol + "/s/" + date
		contents, err := ioutil.ReadFile(stock_file)
		if err != nil {
			//fmt.Printf("Could not read stock_file! err: %s", err)
			errors = append(errors, err)
			continue
		}
		cleanupStockFile(stock_file, contents)
	}
	if len(errors) == 0 {
		errors = nil
	}
	return errors
}

func cleanupOptionFile(fileName string, contents []byte) {
	suspectFileName := fileName + ".suspect"
	cleanFileName := fileName + ".clean"
	fmt.Printf("CLEANUP OPTIONS!!\n\t%s\n\t%s\n\t%s\n", fileName, suspectFileName, cleanFileName)
}

func cleanupStockFile(fileName string, contents []byte) {
	suspectFileName := fileName + ".suspect"
	cleanFileName := fileName + ".clean"
	fmt.Printf("CLEANUP STOCK!!\n\t%s\n\t%s\n\t%s\n", fileName, suspectFileName, cleanFileName)
}

func (c *collector) Collect(symbol string, pipe chan bool) {
	now := time.Now().UTC()
	filename := now.Format("20060102")
	timestamp := now.Format("150405")
	logpath := fmt.Sprintf("%s/log", c.root_dir)

	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		message := "No need for Sat, Sun."
		lazyAppendFile(logpath, filename, timestamp+" : "+message)
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
		lazyAppendFile(logpath, filename, timestamp+" : "+message)
		fmt.Println(message)
		pipe <- false
		return
	}

	limit := now.AddDate(0, 0, 22).Format("20060102")

	options, stock, err := c.adapter.GetOptions(symbol)
	if err != nil {
		message := fmt.Sprintf("Got err: %s", err)
		lazyAppendFile(logpath, filename, timestamp+" : "+message)
		fmt.Println(message)
		pipe <- false
		return
	}

	es, _ := funcs.Encode(&stock, funcs.StockEncodingOrder)

	path := fmt.Sprintf("%s/data/%s/s", c.root_dir, stock.Symbol)
	lazyAppendFile(path, filename, timestamp+","+es)

	for _, option := range options {
		if option.Expiration > limit {
			continue
		}
		eo, err := funcs.Encode(&option, funcs.OptionEncodingOrder)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		path := fmt.Sprintf("%s/data/%s/o/%s/%s", c.root_dir, option.Underlying, option.Expiration, option.Type)
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
