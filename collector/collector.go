package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Collector struct {
	root_dir string
	Adapter  *tdameritrade.TDAmeritrade
}

func New(root_dir string) *Collector {
	return &Collector{root_dir: root_dir}
}

func (c *Collector) Clean(date string) []error {
	errors := []error{}

	if date == "yesterday" {
		// Get 'yymmdd' for yesterday.
		date = time.Now().UTC().AddDate(0, 0, -1).Format("20060102")
	}

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
			if expiration < date {
				fmt.Printf("No need to check exp: %s, date: %s\n", expiration, date)
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
				cleanFile(cleanup_file, contents, "option")
			}
		}
		stock_file := data_dir + "/" + symbol + "/s/" + date
		contents, err := ioutil.ReadFile(stock_file)
		if err != nil {
			//fmt.Printf("Could not read stock_file! err: %s", err)
			errors = append(errors, err)
			continue
		}
		cleanFile(stock_file, contents, "stock")
	}
	if len(errors) == 0 {
		errors = nil
	}
	return errors
}

func cleanFile(fileName string, contents []byte, _type string) {
	suspectFilename := fileName + ".suspect"
	cleanFilename := fileName + ".clean"

	randomExt := fmt.Sprintf("%d", rand.Intn(100000))
	os.Rename(suspectFilename, suspectFilename+"."+randomExt)
	os.Rename(cleanFilename, cleanFilename+"."+randomExt)

	lines := bytes.Split(contents, []byte("\n"))
	good := 0
	bad := 0
	for _, line := range lines {
		rows := bytes.Split(line, []byte(","))
		if len(rows) < 3 {
			continue
		}
		equity := string(bytes.Join(rows[1:], []byte(",")))
		time2 := "000001"
		if _type == "stock" {
			s := structs.Stock{}
			funcs.Decode(equity, &s, funcs.StockEncodingOrder)
			time2 = s.Time
		} else if _type == "option" {
			o := structs.Option{}
			funcs.Decode(equity, &o, funcs.OptionEncodingOrder)
			time2 = o.Time
		}
		near, diff := isNear(string(rows[0]), time2)
		if near {
			good += 1
			lazyAppendFile(filepath.Dir(cleanFilename), filepath.Base(cleanFilename), string(line))
		} else {
			if math.Abs(diff) < 60 {
				//fmt.Printf("%s: %d, %s\n", string(rows[1]), int(diff), string(rows[0]))
			}

			bad += 1
			lazyAppendFile(filepath.Dir(suspectFilename), filepath.Base(suspectFilename), string(line))
		}
	}
	//fmt.Printf("%s\n\tgood: %d, bad: %d\n", fileName, good, bad)

	err := os.Rename(cleanFilename, fileName)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Collector) Collect(symbol string, pipe chan bool) {
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

	options, stock, err := c.Adapter.GetOptions(symbol)
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

func isNear(time1 string, time2 string) (bool, float64) {
	now := time.Now().UTC()
	datetime1, _ := time.Parse("20060102 150405", now.Format("20060102")+" "+time1)
	datetime2, _ := time.Parse("20060102 150405", now.Format("20060102")+" "+time2)
	secondsDiff := datetime1.Sub(datetime2).Seconds()

	EST_diff := float64(18000) // -5 hours in seconds.
	EDT_diff := float64(14400) // -4 hours in seconds.
	padding := float64(45)     // Allow time to be within 45 seconds of current time.

	distanceFromEST := math.Abs(secondsDiff - EST_diff)
	distanceFromEDT := math.Abs(secondsDiff - EDT_diff)

	minDiff := distanceFromEST
	if minDiff > distanceFromEDT {
		minDiff = distanceFromEDT
	}

	if minDiff <= padding {
		return true, minDiff
	}

	return false, minDiff
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
