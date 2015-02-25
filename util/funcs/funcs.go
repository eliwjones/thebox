package funcs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var MaximumEncodingOrder = []string{"Timestamp", "OptionType", "Strike", "Underlying", "OptionSymbol", "UnderlyingBid", "OptionAsk", "MaximumBid", "MaxTimestamp"}
var OptionEncodingOrder = []string{"Underlying", "Symbol", "Expiration", "Time", "Strike", "Bid", "Ask", "Last", "Volume", "OpenInterest", "IV", "Type"}
var StockEncodingOrder = []string{"Symbol", "Time", "Bid", "Ask", "Last", "High", "Low", "Volume"}

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }

func ChooseMFromN(m int, n int) []int {
	rand.Seed(time.Now().UnixNano())

	bag := []int{}
	chosen := []int{}
	for i := 0; i < n; i++ {
		bag = append(bag, i)
	}
	if m >= n {
		// Probably too clever and unnecessary..
		// but, if one wants more than there is, one gets all there is.
		return bag
	}
	for i := 0; i < m; i++ {
		c := rand.Intn(len(bag))
		chosen = append(chosen, bag[c])
		bag = append(bag[:c], bag[c+1:]...)
	}
	return chosen
}

func ClockTimeInSeconds(hhmmss string) int64 {
	t, _ := time.Parse("150405", hhmmss)
	return t.Unix() - t.Truncate(24*time.Hour).Unix()
}

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

func ID(underlying string, weeksBack int, multiplier float64, realTime bool) string {
	rand.Seed(time.Now().UnixNano())

	id := fmt.Sprintf("%s_%02d_%.2f", underlying, weeksBack, multiplier)
	randomN := rand.Intn(1000)
	if realTime {
		id += fmt.Sprintf("_realtime_%04d", randomN)
	} else {
		id += fmt.Sprintf("_%d_%04d", time.Now().Unix(), randomN)
	}
	return id
}

func LastSunday(t time.Time) time.Time {
	distance := int(time.Sunday) - int(t.Weekday())

	return t.AddDate(0, 0, distance)
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

func Multiplier(bid int, ask int, commission float64) float64 {
	return float64(bid) / (float64(ask) + commission)
}

func NextFriday(t time.Time) time.Time {
	distance := int(time.Friday) - int(t.Weekday())
	if distance < 0 {
		distance += 7
	}
	return t.AddDate(0, 0, distance)
}

func TimestampID(timestamp int64) int64 {
	// How many seconds into the week are we?
	return timestamp - WeekID(timestamp)
}

func UpdateConfig(path string, lines []string) error {
	f := []byte(strings.Join(lines, "\n"))
	err := ioutil.WriteFile(path, f, 0777)
	return err
}

func WeekID(timestamp int64) int64 {
	sunday := LastSunday(time.Unix(timestamp, 0).UTC())
	return sunday.Truncate(24 * time.Hour).Unix()
}
