package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"time"
)

type Collector struct {
	Reckless bool

	livedir   string
	logdir    string
	errordir  string
	pipe      chan structs.Message
	replies   chan interface{}
	rootdir   string
	symbols   []string
	targets   map[string]map[string]target // "current", "next" for each SYMBOL.
	timestamp string
	Adapter   *tdameritrade.TDAmeritrade
}

type target struct {
	Timestamp int64 // Seconds since epoch target (10 min increments)
	Stock     structs.Stock
	Options   map[string]structs.Option // Keyed by option symbol.
}

func New(rootdir string) *Collector {
	c := &Collector{}

	c.rootdir = rootdir
	c.livedir = fmt.Sprintf("%s/live", rootdir) // /data, /targets, /timestamp ??
	c.logdir = fmt.Sprintf("%s/log", rootdir)
	c.errordir = fmt.Sprintf("%s/error", rootdir)

	c.pipe = make(chan structs.Message, 10000)
	c.replies = make(chan interface{}, 1000)
	c.symbols = []string{}

	c.targets = map[string]map[string]target{}
	c.targets["current"] = map[string]target{}
	c.targets["next"] = map[string]target{}

	c.timestamp = fmt.Sprintf("%d", time.Now().UTC().Unix()-time.Now().UTC().Truncate(24*time.Hour).Unix())

	return c
}

func (c *Collector) RunOnce() {
	// Deserialize livedir+"/now" info into nowData struct.
	c.targets = c.loadTargets()

	// Fire off go collect(symbol) for []symbols
	for _, symbol := range c.symbols {
		go c.collect(symbol)
	}

	// Process messages..
	go func() {
		for message := range c.pipe {
			// nil Data implies all messages have been sent for SYMBOL.
			if message.Data == nil {
				message.Reply <- true
				continue
			}

			// Save to log of all things
			yymmdd, line := c.SaveToLog(message.Data)

			// Update Target struct.
			c.updateTarget(yymmdd, line)
		}
	}()

	// Block until done.
	for _, _ = range c.symbols {
		<-c.replies
	}

	// cycle Targets if necessary.
	c.maybeCycleTargets(time.Now().UTC().Unix())

	// Serialize c.targets to disk.
	c.dumpTargets()
}

func (c *Collector) ProcessStream(start string, end string) {
	sorted_days := []string{}
	current := start
	t, err := time.Parse("20060102", current)
	if err != nil {
		panic(err)
	}
	for {
		if current > end {
			break
		}
		sorted_days = append(sorted_days, current)
		for {
			// Seek to next M-F.  Please refactor to be less ugly.
			t = t.AddDate(0, 0, 1)
			if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
				continue
			}
			break
		}
		current = t.Format("20060102")
	}

	current_timestamp := int64(-1)
	for _, yymmdd := range sorted_days {
		fmt.Println("******************************" + yymmdd + "*******************************")
		log_data, err := ioutil.ReadFile(c.logdir + "/" + yymmdd)
		if err != nil {
			fmt.Println(err)
			continue
		}

		lines := bytes.Split(log_data, []byte("\n"))
		for _, line := range lines {

			log_timestamp := c.updateTarget(yymmdd, string(line))
			if current_timestamp == -1 {
				current_timestamp = log_timestamp
			}
			if log_timestamp == current_timestamp {
				continue
			}
			if log_timestamp == -1 {
				continue
			}
			// log_timestamp has surpassed current_timestamp.
			c.maybeCycleTargets(log_timestamp)

			current_timestamp = log_timestamp
			fmt.Printf("[%s][current_timestamp] %d\n", yymmdd, current_timestamp)
		}
	}
	c.dumpTargets()

}

func (c *Collector) collect(symbol string) {
	options, stock, err := c.Adapter.GetOptions(symbol)

	// Isn't technically safe to write here.. but.. I can stand to lose one error in a race.
	if err != nil {
		message := fmt.Sprintf("Got err: %s", err)
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
		fmt.Println(message)
		c.replies <- false
		return
	}
	// May regret this ugly seeming structure.
	c.pipe <- structs.Message{Data: stock}

	limit := time.Now().AddDate(0, 0, 22).Format("20060102")
	for _, option := range options {
		if option.Expiration > limit {
			continue
		}
		m := structs.Message{Data: option}
		c.pipe <- m
	}
	// Send empty message with c.replies channel to signal done.
	m := structs.Message{Reply: c.replies}
	c.pipe <- m
}

func (c *Collector) Collect(symbol string) error {
	if c.Reckless {
		c.symbols = append(c.symbols, symbol)
		return nil
	}

	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		message := "No need for Sat, Sun."
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
		return fmt.Errorf(message)
	}
	early := "13:28"
	late := "21:02"
	tooEarly, _ := time.Parse("20060102 15:04", time.Now().Format("20060102")+" "+early)
	tooLate, _ := time.Parse("20060102 15:04", time.Now().Format("20060102")+" "+late)
	// Hamfisted block before 13:30 UTC and after 21:00 UTC.
	if time.Now().Before(tooEarly) || time.Now().After(tooLate) {
		message := fmt.Sprintf("Time %s is before %s UTC or after %s UTC", time.Now().Format("15:04:05"), early, late)
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
		return fmt.Errorf(message)
	}

	c.symbols = append(c.symbols, symbol)

	return nil
}

func (c *Collector) dumpTargets() {
	for _type, symboldata := range c.targets {
		for symbol, data := range symboldata {
			d, err := json.Marshal(data)
			if err != nil {
				panic(err)
			}
			path := c.livedir + "/targets/" + _type
			filename := symbol
			err = funcs.LazyWriteFile(path, filename, d)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (c *Collector) loadTargets() map[string]map[string]target {
	targets := map[string]map[string]target{}
	for _type, _ := range c.targets {
		targets[_type] = map[string]target{}

		path := c.livedir + "/targets/" + _type
		entries, err := ioutil.ReadDir(path)
		if err != nil {
			//panic(err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			symbol := entry.Name()
			data, err := ioutil.ReadFile(path + "/" + symbol)
			if err != nil {
				panic(err)
			}
			var t target
			json.Unmarshal(data, &t)
			targets[_type][symbol] = t
		}
	}

	return targets
}

func (c *Collector) maybeCycleTargets(current_timestamp int64) {
	distance_from_target_time := float64(-1)
	closeness := float64(-1)

	interval := getTenMinTimestamp(current_timestamp)
	interval_hhmmss := interval % int64(24*60*60)

	for symbol, _ := range c.targets["current"] {
		// Zero Timestamp suggests there is nothing to promote.
		if c.targets["current"][symbol].Timestamp == 0 {
			continue
		}
		// Old target was promoted, waiting for current interval before doing anything.
		if interval < c.targets["current"][symbol].Timestamp {
			continue
		}
		// Can only potentially skip promotion if we are in Target Interval.
		if interval == c.targets["current"][symbol].Timestamp {
			distance_from_target_time = float64(c.targets["current"][symbol].Timestamp - current_timestamp)
			_, closeness = isNear(interval_hhmmss, c.targets["current"][symbol].Stock.Time, 45)

			// Still a chance of getting closer data. (target is in the future)
			if distance_from_target_time > 0 {
				continue
			}
			// Still a chance of getting closer data.
			// But only care to try for this if data is further than 30 seconds from target.
			if closeness > 30 && math.Abs(distance_from_target_time) < closeness {
				continue
			}
		}

		c.promoteTarget(c.targets["current"][symbol])

		// In here, will want to set next target Timestamp manually if isn't already there.
		// Which, it probably will not be since "acceptable" data is most likely got before interval changeover.
		c.targets["current"][symbol] = c.targets["next"][symbol]
		if c.targets["current"][symbol].Timestamp == 0 {
			c.targets["current"][symbol] = target{Timestamp: interval + int64(10*60), Options: map[string]structs.Option{}}
		}
		c.targets["next"][symbol] = target{}
	}
}

func (c *Collector) promoteTarget(t target) {
	if t.Stock.Symbol == "" {
		message := fmt.Sprintf("promoteTarget err: Empty Target, Discarding, t.Timestampe: %d", t.Timestamp)
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
		return
	}

	yymmdd_in_seconds, lines, err := encodeTarget(t)
	if err != nil {
		message := fmt.Sprintf("promoteTarget err: %s", err)
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
		return
	}
	funcs.LazyAppendFile(c.livedir+"/data", yymmdd_in_seconds, lines)

	// touch appropriate /live/timestamp/<ts> filename.
	ts := fmt.Sprintf("%d", t.Timestamp)
	err = funcs.LazyTouchFile(c.livedir+"/timestamp", ts)
	if err != nil {
		message := fmt.Sprintf("promoteTarget err: %s", err)
		funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
	}
}

func (c *Collector) SaveToLog(message interface{}) (string, string) {
	line := c.timestamp

	switch message.(type) {
	case structs.Stock:
		s, _ := message.(structs.Stock)
		es, _ := funcs.Encode(&s, funcs.StockEncodingOrder)
		line += ",s," + es
	case structs.Option:
		o, _ := message.(structs.Option)
		eo, _ := funcs.Encode(&o, funcs.OptionEncodingOrder)
		line += ",o," + eo
	default:
		panic("SaveToLog switching wrong!")
	}
	// Write to YYMMDD file in logdir.
	filename := time.Now().Format("20060102")
	funcs.LazyAppendFile(c.logdir, filename, line)

	return filename, line
}

func (c *Collector) updateTarget(yymmdd string, line string) int64 {
	t, _ := time.Parse("20060102", yymmdd)
	yymmdd_in_seconds := t.Unix()

	columns := strings.Split(line, ",")
	if len(columns) < 3 {
		return -1
	}

	hhmmss_in_seconds, err := strconv.ParseInt(columns[0], 10, 64)
	if err != nil {
		panic(err)
	}

	utc_timestamp := yymmdd_in_seconds + hhmmss_in_seconds

	_type := columns[1]
	equity := strings.Join(columns[2:], ",")

	switch _type {
	case "s":
		s := structs.Stock{}
		funcs.Decode(equity, &s, funcs.StockEncodingOrder)

		utc_timestamp = c.updateStockTarget(s, utc_timestamp)
	case "o":
		o := structs.Option{}
		funcs.Decode(equity, &o, funcs.OptionEncodingOrder)

		utc_timestamp = c.updateOptionTarget(o, utc_timestamp)
	default:
		panic("Collapse switching wrong!")
	}

	return utc_timestamp
}

func (c *Collector) updateOptionTarget(o structs.Option, utc_timestamp int64) int64 {
	hhmmss_in_seconds := utc_timestamp % int64(24*60*60)
	near, _ := isNear(hhmmss_in_seconds, o.Time, 45)
	if !near {
		return -1
	}

	utc_interval := getTenMinTimestamp(utc_timestamp)
	current_target := c.targets["current"][o.Underlying]
	target_hhmmss := current_target.Timestamp % int64(24*60*60)

	switch {
	case 0 == current_target.Timestamp:
		current_target.Timestamp = utc_interval
		current_target.Options = map[string]structs.Option{}
		current_target.Options[o.Symbol] = o

		c.targets["current"][o.Underlying] = current_target
	case utc_interval == current_target.Timestamp:
		_, new_distance := isNear(target_hhmmss, o.Time, 45)
		_, old_distance := isNear(target_hhmmss, current_target.Options[o.Symbol].Time, 45)

		if new_distance < old_distance {
			current_target.Options[o.Symbol] = o
			c.targets["current"][o.Underlying] = current_target
		}
	case utc_interval < current_target.Timestamp:
		// interval is old, do not want.
		break
	case utc_interval > current_target.Timestamp:
		// Future interval, stick into "next"
		next_target := c.targets["next"][o.Underlying]
		next_target.Timestamp = utc_interval
		next_target.Options = map[string]structs.Option{}
		next_target.Options[o.Symbol] = o

		c.targets["next"][o.Underlying] = next_target
	}

	return utc_timestamp
}

func (c *Collector) updateStockTarget(s structs.Stock, utc_timestamp int64) int64 {
	hhmmss_in_seconds := utc_timestamp % int64(24*60*60)
	near, _ := isNear(hhmmss_in_seconds, s.Time, 45)
	if !near {
		return -1
	}

	utc_interval := getTenMinTimestamp(utc_timestamp)
	current_target := c.targets["current"][s.Symbol]
	target_hhmmss := current_target.Timestamp % int64(24*60*60)

	switch {
	case 0 == current_target.Timestamp:
		current_target.Timestamp = utc_interval
		current_target.Stock = s
		current_target.Options = map[string]structs.Option{}

		c.targets["current"][s.Symbol] = current_target
	case utc_interval == current_target.Timestamp:
		_, new_distance := isNear(target_hhmmss, s.Time, 45)
		_, old_distance := isNear(target_hhmmss, current_target.Stock.Time, 45)

		if new_distance < old_distance {
			current_target.Stock = s
			c.targets["current"][s.Symbol] = current_target
		}
	case utc_interval < current_target.Timestamp:
		// interval is old, do not want.
		break
	case utc_interval > current_target.Timestamp:
		next_target := c.targets["next"][s.Symbol]
		next_target.Timestamp = utc_interval
		next_target.Stock = s
		next_target.Options = map[string]structs.Option{}

		c.targets["next"][s.Symbol] = next_target
	}

	return utc_timestamp
}

func encodeTarget(t target) (string, string, error) {
	yymmdd_in_seconds := t.Timestamp - (t.Timestamp % (24 * 60 * 60))
	hhmmss_in_seconds := t.Timestamp - yymmdd_in_seconds

	es, err := funcs.Encode(&t.Stock, funcs.StockEncodingOrder)
	if err != nil {
		return "", "", err
	}
	lines := fmt.Sprintf("%d,s,%s", hhmmss_in_seconds, es)
	for _, o := range t.Options {
		eo, err := funcs.Encode(&o, funcs.OptionEncodingOrder)
		if err != nil {
			return "", "", err
		}
		lines += fmt.Sprintf("\n%d,o,%s", hhmmss_in_seconds, eo)
	}
	return fmt.Sprintf("%d", yymmdd_in_seconds), lines, nil
}

func getTenMinTimestamp(timestamp int64) int64 {
	ten_min_in_seconds := int64(10 * 60)
	r := timestamp % ten_min_in_seconds
	before := timestamp - r
	between := before + ten_min_in_seconds/2
	after := before + ten_min_in_seconds

	if timestamp < between {
		return before
	}
	return after

}

func isNear(utc_time int64, local_time int64, padding int) (bool, float64) {
	secondsDiff := float64(utc_time) - float64(local_time)

	EST_diff := float64(18000) // -5 hours in seconds.
	EDT_diff := float64(14400) // -4 hours in seconds.

	distanceFromEST := math.Abs(secondsDiff - EST_diff)
	distanceFromEDT := math.Abs(secondsDiff - EDT_diff)

	minDiff := distanceFromEST
	if minDiff > distanceFromEDT {
		minDiff = distanceFromEDT
	}

	if minDiff <= float64(padding) {
		return true, minDiff
	}

	return false, minDiff
}
