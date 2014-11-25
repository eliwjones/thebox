package funcs

import (
	"github.com/eliwjones/thebox/util/structs"

	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var OptionEncodingOrder = []string{"Underlying", "Symbol", "Expiration", "Time", "Strike", "Bid", "Ask", "Last", "Volume", "OpenInterest", "IV", "Type"}

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }

func DecodeOption(eo string) (structs.Option, error) {
	o := structs.Option{}
	r := reflect.ValueOf(&o).Elem()

	s := strings.Split(eo, ",")
	for idx, v := range s {
		f := r.FieldByName(OptionEncodingOrder[idx])
		switch f.Type().Kind() {
		case reflect.String:
			f.SetString(v)
		case reflect.Int:
			val, _ := strconv.ParseInt(v, 10, 0)
			f.SetInt(val)
		case reflect.Float64:
			val, _ := strconv.ParseFloat(v, 64)
			f.SetFloat(val)

		}
	}

	return o, nil
}

func EncodeOption(o structs.Option) (string, error) {
	r := reflect.ValueOf(&o).Elem()
	eo := ""
	for _, propname := range OptionEncodingOrder {
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

func GetConfig() (string, string, string, string, error) {
	b, err := ioutil.ReadFile("config")
	if err != nil {
		return "", "", "", "", err
	}
	lines := strings.Split(string(b), "\n")
	return lines[0], lines[1], lines[2], lines[3], nil
}

func UpdateConfig(id string, pass string, sid string, jsess string) error {
	f := []byte(fmt.Sprintf("%s\n%s\n%s\n%s", id, pass, sid, jsess))
	err := ioutil.WriteFile("config", f, 0777)
	return err
}
