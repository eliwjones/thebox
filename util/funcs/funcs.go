package funcs

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var OptionEncodingOrder = []string{"Underlying", "Symbol", "Expiration", "Time", "Strike", "Bid", "Ask", "Last", "Volume", "OpenInterest", "IV", "Type"}
var StockEncodingOrder = []string{"Symbol", "Time", "Bid", "Ask", "Last", "High", "Low", "Volume"}

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }

func Decode(eo string, c interface{}, encodingOrder []string) error {
	r := reflect.ValueOf(c).Elem()

	s := strings.Split(eo, ",")
	if len(s) != len(encodingOrder) {
		return fmt.Errorf("Expected %d Items.  Got %d Items.", len(encodingOrder), len(s))
	}
	for idx, v := range s {
		f := r.FieldByName(encodingOrder[idx])
		switch f.Type().Kind() {
		case reflect.String:
			f.SetString(v)
		case reflect.Int:
			val, err := strconv.ParseInt(v, 10, 0)
			if err != nil {
				return err
			}
			f.SetInt(val)
		case reflect.Int64:
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			f.SetInt(int64(val))
		case reflect.Float64:
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			f.SetFloat(val)

		}
	}

	return nil
}

func Encode(c interface{}, encodingOrder []string) (string, error) {
	r := reflect.ValueOf(c).Elem()

	eo := ""
	for _, propname := range encodingOrder {
		v := r.FieldByName(propname)
		switch v.Type().Kind() {
		case reflect.String:
			eo += "," + v.String()
		case reflect.Int, reflect.Int64:
			eo += "," + fmt.Sprintf("%d", v.Int())
		case reflect.Float64:
			eo += "," + fmt.Sprintf("%.5f", v.Float())

		}
	}
	if strings.Count(eo[1:], ",")+1 != len(encodingOrder) {
		return eo[1:], fmt.Errorf("Expected %d Items.  Got %d Items.", len(encodingOrder), strings.Count(eo[1:], ",")+1)
	}
	return eo[1:], nil
}

func GetConfig(path string) ([]string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}
	lines := strings.Split(string(b), "\n")
	return lines, nil
}

func LazyAppendFile(folderName string, fileName string, data string) error {
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

func LazyTouchFile(folderName string, fileName string) error {
	f, err := os.OpenFile(folderName+"/"+fileName, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		os.MkdirAll(folderName, 0777)
		f, err = os.OpenFile(folderName+"/"+fileName, os.O_RDWR|os.O_CREATE, 0777)
	}
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func LazyWriteFile(folderName string, fileName string, data []byte) error {
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

func ClockTimeInSeconds(hhmmss string) int64 {
	t, _ := time.Parse("150405", hhmmss)
	return t.Unix() - t.Truncate(24*time.Hour).Unix()
}

func UpdateConfig(path string, lines []string) error {
	f := []byte(strings.Join(lines, "\n"))
	err := ioutil.WriteFile(path, f, 0777)
	return err
}
