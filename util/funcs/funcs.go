package funcs

import (
	"fmt"
	"io/ioutil"
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
		case reflect.Int:
			eo += "," + fmt.Sprintf("%d", v.Int())
		case reflect.Float64:
			eo += "," + fmt.Sprintf("%.5f", v.Float())

		}
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

func UpdateConfig(path string, lines []string) error {
	f := []byte(strings.Join(lines, "\n"))
	err := ioutil.WriteFile(path, f, 0777)
	return err
}
