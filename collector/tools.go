package collector

import (
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// These are more like helper functions and not part of Collector.  Thus, detaching from main struct.
func Clean(root_dir string, date string) []error {
	errors := []error{}

	if date == "yesterday" {
		// Get 'yymmdd' for yesterday.
		date = time.Now().UTC().AddDate(0, 0, -1).Format("20060102")
	}

	data_dir := root_dir + "/data"
	d, err := os.Open(data_dir)
	if err != nil {
		errors = append(errors, err)
	}
	defer d.Close()
	symbols, _ := d.Readdirnames(-1)
	if len(symbols) == 0 {
		errors = append(errors, fmt.Errorf("Nothing to clean!"))
	}
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

	rows := bytes.Split(contents, []byte("\n"))
	good := 0
	bad := 0
	for _, row := range rows {
		columns := bytes.Split(row, []byte(","))
		if len(columns) < 3 {
			continue
		}
		equity := string(bytes.Join(columns[1:], []byte(",")))
		time2 := int64(1)
		if _type == "stock" {
			s := structs.Stock{}
			funcs.Decode(equity, &s, funcs.StockEncodingOrder)
			time2 = s.Time
		} else if _type == "option" {
			o := structs.Option{}
			funcs.Decode(equity, &o, funcs.OptionEncodingOrder)
			time2 = o.Time
		}
		time1, err := strconv.ParseFloat(string(columns[0]), 64)
		if err != nil {
			panic(err)
		}
		near, _ := isNear(int64(time1), time2, 45)
		if near {
			good += 1
			funcs.LazyAppendFile(filepath.Dir(cleanFilename), filepath.Base(cleanFilename), string(row))
		} else {
			bad += 1
			funcs.LazyAppendFile(filepath.Dir(suspectFilename), filepath.Base(suspectFilename), string(row))
		}
	}

	if good > 0 {
		err := os.Rename(cleanFilename, fileName)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		// Seems inefficient, but this will only happen for a week day with no valid data.
		// Presumably, should happen only on Holidays?  Though, those should technically have one "valid" line.
		err := os.Rename(fileName, suspectFilename)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func Migrate(root_dir string, date string) []error {
	errors := []error{}

	data_dir := root_dir + "/data"
	d, _ := os.Open(data_dir)
	defer d.Close()
	symbols, _ := d.Readdirnames(-1)
	for _, symbol := range symbols {
		options_dir := data_dir + "/" + symbol + "/o"
		e, err := os.Open(options_dir)
		if err != nil {
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
					errors = append(errors, err)
					continue
				}
				migrateFile(cleanup_file, contents, "option")
			}
		}
		stock_file := data_dir + "/" + symbol + "/s/" + date
		contents, err := ioutil.ReadFile(stock_file)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		migrateFile(stock_file, contents, "stock")
	}
	if len(errors) == 0 {
		errors = nil
	}
	return errors
}

func migrateFile(fileName string, contents []byte, _type string) {
	migratedFilename := fileName + ".migrated"

	randomExt := fmt.Sprintf("%d", rand.Intn(100000))
	os.Rename(migratedFilename, migratedFilename+"."+randomExt)

	rows := bytes.Split(contents, []byte("\n"))
	for _, row := range rows {
		columns := bytes.Split(row, []byte(","))
		if len(columns) < 3 {
			continue
		}
		// Convert column[0] to Seconds
		t, _ := time.Parse("150405", string(columns[0]))
		converted_time := fmt.Sprintf("%d", t.Unix()-t.Truncate(24*time.Hour).Unix())
		columns[0] = []byte(converted_time)
		if _type == "stock" {
			t, _ = time.Parse("150405", string(columns[2]))
			converted_time = fmt.Sprintf("%d", t.Unix()-t.Truncate(24*time.Hour).Unix())
			columns[2] = []byte(converted_time)
		} else if _type == "option" {
			t, _ = time.Parse("150405", string(columns[4]))
			converted_time = fmt.Sprintf("%d", t.Unix()-t.Truncate(24*time.Hour).Unix())
			columns[4] = []byte(converted_time)
		}
		row = bytes.Join(columns, []byte(","))
		funcs.LazyAppendFile(filepath.Dir(migratedFilename), filepath.Base(migratedFilename), string(row))
	}

	//err := os.Rename(fileName, fileName + ".original")
	err := os.Rename(migratedFilename, fileName)
	if err != nil {
		fmt.Println(err)
	}
}
