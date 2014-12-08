package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
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
	// Get list of /log files for day range.
	sorted_days := []string{}
	for _, day := range sorted_days {
		// load log file.
		yymmdd := "day_ts to yymmdd" + day
		current_timestamp := int64(-1)
		lines := []string{}
		for _, line := range lines {
			log_timestamp, _ := c.updateTarget(yymmdd, line)
			if current_timestamp == -1 {
				current_timestamp = log_timestamp
			}
			if log_timestamp == current_timestamp {
				continue
			}
			// log_timestamp has surpassed current_timestamp.
			c.maybeCycleTargets(log_timestamp)

			current_timestamp = log_timestamp
		}
	}

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
	for symbol, _ := range c.targets["current"] {
		distance := current_timestamp - c.targets["current"][symbol].Timestamp
		if distance < 0 {
			// Target is in the future, do not persist.
			continue
		}

		c.promoteTarget(c.targets["current"][symbol])

		// cycle "next" -> "current"
		c.targets["current"][symbol] = c.targets["next"][symbol]

		// initialize "next" for 10 minutes in the future.
		next_timestamp := c.targets["next"][symbol].Timestamp + 10*60
		c.targets["next"][symbol] = target{Timestamp: next_timestamp}
	}
}

func (c *Collector) promoteTarget(t target) {
	// Send target to /live/data/yymmdd_seconds.<>
	// Encode all Stock and Option data and lazyAppendFile() gigantic multiline blob.
	
	// touch appropriate /live/timestamp/<ts> filename.
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

func (c *Collector) updateTarget(yymmdd string, line string) (int64, string) {
	// Update "current" and possibly "next" targets as needed.
	// return fully constructed timestamp for log line.

	// When updating targets.  If target.Timestamp == 0,
	//    seek to next 2 valid 10 min timestamps.
	//    set "current" and "next" Timestamps to these values.

	// Next valid timestamp is..? somefunction of log line Time?
	// In this case, do isNear() check to make sure log line is reasonably valid.

	// If isNear(), round down if within 4 mins of 10 min window., else round up to next 10 min window..

	line_timestamp := int64(0)
	return line_timestamp, "SYMBOL"
	/*
		t, _ := time.Parse("20060102", yymmdd)
		yymmdd_in_seconds := t.Unix()

		// Split out columns
		columns := strings.Split(line, ",")
		hhmmss_in_seconds, err := strconv.ParseInt(columns[0], 10, 64)
		if err != nil {
			panic(err)
		}

		timestamp := yymmdd_in_seconds + hhmmss_in_seconds

		_type := columns[1]
		equity := strings.Join(columns[2:], ",")

		equity_time := int64(0)
		switch _type {
		case "s":
			s := structs.Stock{}
			funcs.Decode(equity, &s, funcs.StockEncodingOrder)

			c.UpdateCurrentStock(timestamp, s)
			equity_time = s.Time
		case "o":
			o := structs.Option{}
			funcs.Decode(equity, &o, funcs.OptionEncodingOrder)

			c.UpdateCurrentOptionChain(timestamp, o)
			equity_time = o.Time
		default:
			panic("Collapse switching wrong!")
		}

		near, _ := isNear(equity_time, hhmmss_in_seconds, int64(45))
		if !near {
			// Stale stock, option info so not using.
			return
		}
		// target_time may be defined in current_dir.
		// If not there, we are waiting for 09:30:00
		target_time := funcs.ClockTimeInSeconds("930")
		last_distance := int64(45)
		near, distance := isNear(hhmmss_in_seconds, target_time, last_distance)
		if !near {
			// Not closer to target_time than last run.
			//
		}
	*/
}

func isNear(time1 int64, time2 int64, padding int64) (bool, float64) {
	secondsDiff := float64(time1) - float64(time2)

	EST_diff := float64(18000) // -5 hours in seconds.
	EDT_diff := float64(14400) // -4 hours in seconds.

	distanceFromEST := secondsDiff - EST_diff
	distanceFromEDT := secondsDiff - EDT_diff

	minDiff := distanceFromEST
	if math.Abs(minDiff) > math.Abs(distanceFromEDT) {
		minDiff = distanceFromEDT
	}

	if math.Abs(minDiff) <= float64(padding) {
		return true, minDiff
	}

	return false, minDiff
}
