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
	"sort"
	"strconv"
	"strings"
	"time"
)

type Collector struct {
	Reckless bool

	id        string
	livedir   string
	logdir    string
	errordir  string
	maximums  map[string]map[string][]structs.Maximum // keyed off of (Expiration, OptionSymbol)
	period    int64
	pipe      chan structs.Message
	replies   chan interface{}
	rootdir   string
	symbols   []string
	targets   map[string]map[string]target // "current", "next" for each SYMBOL.
	timestamp string
	Adapter   *tdameritrade.TDAmeritrade

	// Most likely ill-advised lazy load structures.
	// But, in time crunch, and can make sexy when have nothing better to do.  Trading is more important.
	getQuotesChan chan getQuotesMessage               // Requests for quotes controlled by this channel.
	quote         map[int64]map[string]structs.Option // For holding individual timestamped quotes. (Trader likes this)
}

// Ugly feeling first pass.
type getQuotesMessage struct {
	timestamp int64
	reply     chan map[string]structs.Option
}

type target struct {
	Timestamp int64 // Seconds since epoch target (10 min increments)
	Stock     structs.Stock
	Options   map[string]structs.Option // Keyed by option symbol.
}

type ByTimestampID []structs.Maximum

func (m ByTimestampID) Len() int      { return len(m) }
func (m ByTimestampID) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m ByTimestampID) Less(i, j int) bool {
	idI := funcs.TimestampID(m[i].Timestamp)
	idJ := funcs.TimestampID(m[j].Timestamp)
	if idI == idJ {
		if m[i].Timestamp == m[j].Timestamp {
			return m[i].OptionType < m[j].OptionType
		}
		return m[i].Timestamp < m[j].Timestamp
	}
	return idI < idJ
}

func New(id string, rootdir string, period int64) *Collector {
	c := &Collector{}

	c.id = id
	c.rootdir = rootdir
	c.livedir = fmt.Sprintf("%s/live", rootdir) // /data, /targets, /timestamp ??
	c.logdir = fmt.Sprintf("%s/log", rootdir)
	c.errordir = fmt.Sprintf("%s/error", rootdir)

	c.period = period
	c.pipe = make(chan structs.Message, 10000)
	c.getQuotesChan = make(chan getQuotesMessage, 100)
	c.quote = map[int64]map[string]structs.Option{}
	c.replies = make(chan interface{}, 1000)
	c.symbols = []string{}

	c.maximums = map[string]map[string][]structs.Maximum{}

	c.targets = map[string]map[string]target{}
	c.targets["current"] = map[string]target{}
	c.targets["next"] = map[string]target{}

	c.timestamp = fmt.Sprintf("%d", time.Now().UTC().Unix()-time.Now().UTC().Truncate(24*time.Hour).Unix())

	// Process GetQuotes() calls for unloaded data.
	go func() {
		for message := range c.getQuotesChan {
			utcTimestamp := message.timestamp
			qs, exists := c.quote[utcTimestamp]
			if !exists {
				quotes, err := c.getQuotes(utcTimestamp)
				if err != nil {
					fmt.Printf("getQuotes() Err: %s\n", err)
				}
				qs = quotes
			}
			message.reply <- qs
		}
	}()

	return c
}

func (c *Collector) RunOnce() {
	// Must panic if RunOnce() takes longer than 50 seconds?
	// Longer than cron period will result in out-of-order log entries.
	startTime := time.Now().UTC().Unix()

	// Deserialize from disk.
	c.targets = c.loadTargets()
	c.maximums = c.loadMaximums()

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
			if time.Now().UTC().Unix()-startTime > (c.period - 10) {
				panicMessage := fmt.Sprintf("We are too close to the collector period of %d seconds. Panicking", c.period)
				c.logError("RunOnce", panicMessage)
				panic(panicMessage)
			}

			// Save to log of all things
			yyyymmdd, line := c.SaveToLog(message.Data)

			logTimestamp, _type, encodedEquity := c.parseLogLine(yyyymmdd, string(line))

			// Update the things.
			o, err := c.updateTarget(logTimestamp, _type, encodedEquity)
			if err == nil {
				c.updateMaximum(o, logTimestamp)
			}
		}
	}()

	// Block until done.
	for _, _ = range c.symbols {
		<-c.replies
	}

	// cycle Targets if necessary.
	// "sadly" addition of new max timestamp is hidden away in promoteTarget().
	// which is inside maybeCycleTargets()
	c.maybeCycleTargets(time.Now().UTC().Unix())
	c.maybeCycleMaximums(time.Now().UTC().Unix())

	// Serialize to disk.
	c.dumpTargets()
	c.dumpMaximums()
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

	currentTimestamp := int64(-1)
	for _, yyyymmdd := range sorted_days {
		fmt.Println("******************************" + yyyymmdd + "*******************************")
		log_data, err := ioutil.ReadFile(c.logdir + "/" + yyyymmdd)
		if err != nil {
			fmt.Println(err)
			continue
		}

		lines := bytes.Split(log_data, []byte("\n"))
		for _, line := range lines {
			logTimestamp, _type, encodedEquity := c.parseLogLine(yyyymmdd, string(line))
			if logTimestamp != currentTimestamp && currentTimestamp != -1 {
				c.maybeCycleTargets(currentTimestamp)
				c.maybeCycleMaximums(currentTimestamp)

				fmt.Printf("[%s][current_timestamp] %d\n", yyyymmdd, currentTimestamp)
			}
			o, err := c.updateTarget(logTimestamp, _type, encodedEquity)
			if err == nil {
				c.updateMaximum(o, logTimestamp)
			}
			currentTimestamp = logTimestamp
		}
		// Just in case fate is frowning on us.  (This should be almost always unnecessary.)
		c.maybeCycleTargets(currentTimestamp)
		c.maybeCycleMaximums(currentTimestamp)

		// Reset targets.. since they don't carry over into new days.
		c.targets["current"] = map[string]target{}
		c.targets["next"] = map[string]target{}
	}
	c.dumpTargets()
	c.dumpMaximums()

}

func (c *Collector) addMaximum(o structs.Option, s structs.Stock, timestamp int64) error {
	// No Tracking for anything farther than 1 week from expiration.
	//    subtracting 6 days in seconds since o.Expiration will parse to 00:00.
	//    no need to be too exact since options don't trade on Sat, Sun.
	// No Tracking for anything past expiration.
	t, _ := time.Parse("20060102", o.Expiration)
	expirationDistance := timestamp - t.Unix()
	if expirationDistance < int64(-6*24*60*60) {
		return fmt.Errorf("Further than 1 week from expiration. Distance: %d", expirationDistance)
	}
	if expirationDistance > int64(1*24*60*60) {
		return fmt.Errorf("Expiration past. Distance: %d", expirationDistance)
	}

	// Filtering out uninteresting Options here.
	if o.Ask <= 0 {
		return fmt.Errorf("Not interested in options with Ask <= 0.")
	}
	// No idea how Volume could be negative, but not interested in finding out.
	if o.Volume <= 0 {
		return fmt.Errorf("Not interested in options with Volume <= 0.")
	}
	// Only want out-of-the-money options.
	if o.Type == "c" && o.Strike < s.Bid {
		return fmt.Errorf("Not interested in out-of-the-money options.")
	}
	if o.Type == "p" && o.Strike > s.Bid {
		return fmt.Errorf("Not interested in out-of-the-money options.")
	}

	m := structs.Maximum{}
	m.Expiration = o.Expiration
	m.MaximumBid = o.Bid
	m.OptionAsk = o.Ask
	m.OptionBid = o.Bid
	m.OptionSymbol = o.Symbol
	m.OptionType = o.Type
	m.Strike = o.Strike
	m.Timestamp = timestamp
	m.Underlying = o.Underlying
	m.UnderlyingBid = s.Bid
	m.Volume = o.Volume
	m.MaxTimestamp = timestamp

	_, expExists := c.maximums[o.Expiration]
	if !expExists {
		c.maximums[o.Expiration] = map[string][]structs.Maximum{}
	}
	_, optionSymbolExists := c.maximums[o.Expiration][o.Symbol]
	if !optionSymbolExists {
		c.maximums[o.Expiration][o.Symbol] = []structs.Maximum{}
	}
	c.maximums[o.Expiration][o.Symbol] = append(c.maximums[o.Expiration][o.Symbol], m)

	return nil
}

func (c *Collector) collect(symbol string) (string, string) {
	thisMonth := time.Now().Format("200601")
	limit := time.Now().AddDate(0, 0, 22)
	limitMonth := limit.Format("200601")

	options, stock, err := c.Adapter.GetOptions(symbol, thisMonth)

	// Isn't technically safe to write here.. but.. I can stand to lose one error in a race.
	if err != nil {
		c.logError("collect ("+thisMonth+") - "+symbol, err)
		fmt.Println(err)
		c.replies <- false
		return thisMonth, limitMonth
	}
	if limitMonth != thisMonth {
		optionsNextMonth, _, err := c.Adapter.GetOptions(symbol, limitMonth)
		if err != nil {
			c.logError("collect ("+limitMonth+") - "+symbol, err)
			c.replies <- false
			return thisMonth, limitMonth
		}
		options = append(options, optionsNextMonth...)
	}

	// May regret this ugly seeming structure.
	c.pipe <- structs.Message{Data: stock}

	for _, option := range options {
		if option.Expiration > limit.Format("20060102") {
			continue
		}
		m := structs.Message{Data: option}
		c.pipe <- m
	}
	// Send empty message with c.replies channel to signal done.
	m := structs.Message{Reply: c.replies}
	c.pipe <- m

	return thisMonth, limitMonth
}

func (c *Collector) Collect(symbol string) error {
	if c.Reckless {
		c.symbols = append(c.symbols, symbol)
		return nil
	}

	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		c.logError("Collect", "No need for Sat, Sun.")
		return fmt.Errorf("No need for Sat, Sun.")
	}
	early := "13:28"
	late := "21:02"
	tooEarly, _ := time.Parse("20060102 15:04", time.Now().Format("20060102")+" "+early)
	tooLate, _ := time.Parse("20060102 15:04", time.Now().Format("20060102")+" "+late)
	// Hamfisted block before 13:30 UTC and after 21:00 UTC.
	if time.Now().Before(tooEarly) || time.Now().After(tooLate) {
		message := fmt.Sprintf("Time %s is before %s UTC or after %s UTC", time.Now().UTC().Format("15:04:05"), early, late)
		c.logError("Collect", message)
		return fmt.Errorf(message)
	}

	c.symbols = append(c.symbols, symbol)

	return nil
}

func (c *Collector) DeserializeMaximums(maximums string) ([]structs.Maximum, error) {
	var e error
	Maximums := []structs.Maximum{}
	for _, maximum := range strings.Split(maximums, "\n") {
		m := structs.Maximum{}
		err := funcs.Decode(maximum, &m, funcs.MaximumEncodingOrder)
		if err != nil {
			fmt.Printf("Maximum: %s, Err: %s\n", maximum, err)
			e = err
		}
		Maximums = append(Maximums, m)
	}
	return Maximums, e
}

func (c *Collector) dumpMaximums() {
	// Not sure of least dumb way to structure data for Marshal, Unmarshal..
	// Expirementing with marshing all directly to current sub-dir with collector.id as filename.
	// This is lazy and wastes space.. but.. looks like its about 67MB per Expiration.  So, I can deal with that for now.
	// Special encoding would squeeze down to 20MB.. but.. that's only 3x so who cares.

	d, err := json.Marshal(c.maximums)
	if err != nil {
		c.logError("dumpMaximums", err)
		return
	}
	path := c.livedir + "/maximums/current"
	filename := c.id
	err = funcs.LazyWriteFile(path, filename, d)
	if err != nil {
		c.logError("dumpMaximums", err)
	}
}

func (c *Collector) dumpTargets() {
	for _type, symboldata := range c.targets {
		for symbol, data := range symboldata {
			d, err := json.Marshal(data)
			if err != nil {
				c.logError("dumpTargets", err)
				continue
			}
			path := c.livedir + "/targets/" + _type
			filename := symbol
			err = funcs.LazyWriteFile(path, filename, d)
			if err != nil {
				c.logError("dumpTargets", err)
			}
		}
	}
}

func (c *Collector) GetPastNEdges(utcTimestamp int64, n int) []structs.Maximum {
	// edgeMap["<Underlying>_<option.type>_<edgeID>"][]structs.Maximum{}
	edgeMap := map[string][]structs.Maximum{}

	pastNFridays := []time.Time{}
	friday := funcs.NextFriday(time.Unix(utcTimestamp, 0).UTC())
	for i := 0; i < n; i++ {
		friday = friday.AddDate(0, 0, -7)
		pastNFridays = append(pastNFridays, friday)
	}

	for _, friday := range pastNFridays {
		expiration := friday.Format("20060102")
		filepath := c.livedir + "/edges/" + expiration
		edgeData, err := ioutil.ReadFile(filepath)
		if err != nil {
			c.logError("getPastNEdges", err)
			// Weekly Expiration may be old Saturday type.
			expiration = friday.AddDate(0, 0, 1).Format("20060102")
			filepath := c.livedir + "/edges/" + expiration
			edgeData, err = ioutil.ReadFile(filepath)
		}
		if err != nil {
			c.logError("getPastNEdges", err)
			continue
		}
		// Have edges.. look for appropriate one.
		edges, _ := c.DeserializeMaximums(string(edgeData))
		for _, e := range edges {
			// Underlying_type_TimestampID
			edgeKey := getEdgeKey(e)
			e.Expiration = expiration
			edgeMap[edgeKey] = append(edgeMap[edgeKey], e)
		}
	}
	// Fill in gaps.
	//    I.e. if edges exist for an Underlying_optionType_edgeID, there must be N of them.
	// Amount of contortions happening is starting to smell bad.
	for edgeKey, edges := range edgeMap {
		m := len(edges)
		if m == n {
			continue
		}
		optionType := getOptionTypeFromEdgeKey(edgeKey)
		tsID := getTimestampIDFromEdgeKey(edgeKey)
		underlyingSymbol := getUnderlyingFromEdgeKey(edgeKey)
		for i := 0; i < n-m; i++ {
			edgeMap[edgeKey] = append(edgeMap[edgeKey], structs.Maximum{Underlying: underlyingSymbol, OptionType: optionType, Timestamp: tsID})
		}
	}

	// Want large list of sorted edges for easy viewing when Encoded.
	pastNEdges := []structs.Maximum{}
	for _, edge := range edgeMap {
		pastNEdges = append(pastNEdges, edge...)
	}
	sort.Sort(ByTimestampID(pastNEdges))
	return pastNEdges
}

func (c *Collector) GetQuote(utcTimestamp int64, underlying string, symbol string) (structs.Option, error) {
	var err error
	quote, exists := c.quote[utcTimestamp][symbol]
	if !exists {
		// Missing c.quote info, must populate all quotes.
		message := getQuotesMessage{timestamp: utcTimestamp}
		message.reply = make(chan map[string]structs.Option)
		c.getQuotesChan <- message
		<-message.reply
		quote, exists = c.quote[utcTimestamp][symbol]
		if !exists {
			err = fmt.Errorf("Symbol: %s does not exist for timestamp: %d", symbol, utcTimestamp)
		}
	}
	return quote, err
}

func (c *Collector) GetQuotes(utcTimestamp int64, underlying string) ([]structs.Option, error) {
	qs, exists := c.quote[utcTimestamp]
	if !exists {
		// Send request down getquotes channel.  Await response.
		message := getQuotesMessage{timestamp: utcTimestamp}
		message.reply = make(chan map[string]structs.Option)
		c.getQuotesChan <- message
		qs = <-message.reply
	}

	// Lazy Filter.
	quotes := []structs.Option{}
	for _, quote := range qs {
		if quote.Underlying != underlying {
			continue
		}
		quotes = append(quotes, quote)
	}

	return quotes, nil
}

func (c *Collector) getQuotes(utcTimestamp int64) (map[string]structs.Option, error) {
	// Now just returns all quotes across all symbols.
	// Will do more clever filtering if required later on.

	// Directly populating c.quote datastructure.

	utcTime := time.Unix(utcTimestamp, 0).UTC()
	yyyymmdd := utcTime.Format("20060102")

	thisFriday := funcs.NextFriday(utcTime).Format("20060102")
	thisSaturday := funcs.NextFriday(utcTime).AddDate(0, 0, 1).Format("20060102")

	// load livedir /quotes/yyyymmdd, decode, and filter by underlying, expiration.
	quoteData, err := ioutil.ReadFile(c.livedir + "/quotes/" + yyyymmdd)
	if err != nil {
		return map[string]structs.Option{}, err
	}

	quotesCopy := map[int64]map[string]structs.Option{}
	for ts, quotes := range c.quote {
		quotesCopy[ts] = quotes
	}

	for _, line := range bytes.Split(quoteData, []byte("\n")) {
		ts, _type, encodedEquity := c.parseQuoteLine(string(line))
		if ts == -1 || _type != "o" {
			continue
		}
		o := structs.Option{}
		funcs.Decode(encodedEquity, &o, funcs.OptionEncodingOrder)
		if o.Underlying == "" {
			continue
		}
		if o.Expiration != thisFriday && o.Expiration != thisSaturday {
			continue
		}

		// Have proper expiration.
		_, exists := quotesCopy[ts]
		if !exists {
			quotesCopy[ts] = map[string]structs.Option{}
		}
		quotesCopy[ts][o.Symbol] = o
	}
	// Presumably this is safe since getQuotesChannel disallows concurrent access.
	c.quote = quotesCopy
	return c.quote[utcTimestamp], nil
}

func (c *Collector) loadMaximums() map[string]map[string][]structs.Maximum {
	maximums := map[string]map[string][]structs.Maximum{}

	path := c.livedir + "/maximums/current"
	data, err := ioutil.ReadFile(path + "/" + c.id)
	if err != nil {
		c.logError("loadMaximums", err)
		return maximums
	}
	// Presumably, we now have last serialized []byte of maximums.
	err = json.Unmarshal(data, &maximums)
	if err != nil {
		c.logError("loadMaximums", err)
	}
	return maximums

}

func (c *Collector) loadTargets() map[string]map[string]target {
	targets := map[string]map[string]target{}
	for _type, _ := range c.targets {
		targets[_type] = map[string]target{}

		path := c.livedir + "/targets/" + _type
		entries, err := ioutil.ReadDir(path)
		if err != nil {
			c.logError("loadTargets", err)
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			symbol := entry.Name()
			data, err := ioutil.ReadFile(path + "/" + symbol)
			if err != nil {
				c.logError("loadTargets", err)
				continue
			}
			var t target
			json.Unmarshal(data, &t)
			targets[_type][symbol] = t
		}
	}

	return targets
}

func (c *Collector) logError(functionName string, err interface{}) {
	err_str := ""
	switch err.(type) {
	case string:
		err_str = err.(string)
	case error:
		e, _ := err.(error)
		err_str = fmt.Sprintf("%s", e)
	}
	message := fmt.Sprintf("[%s] %s", functionName, err_str)
	funcs.LazyAppendFile(c.errordir, time.Now().Format("20060102"), time.Now().Format("15:04:05")+" : "+message)
}

func (c *Collector) maybeCycleMaximums(currentTimestamp int64) {
	yymmdd := time.Unix(currentTimestamp, 0).Format("20060102")
	for expiration, _ := range c.maximums {
		// Not past expiration, do nothing.
		if yymmdd <= expiration {
			continue
		}
		maxpath := c.livedir + "/maximums"
		funcs.LazyWriteFile(maxpath, expiration, []byte(""))
		edges := map[string]structs.Maximum{} // timestamp_symbol_o.Type, maximum

		// Write Edges and Maximums.
		for _, maxSlice := range c.maximums[expiration] {
			for _, max := range maxSlice {
				// Uninterested in non-max maximum.
				if max.MaximumBid <= max.OptionAsk {
					continue
				}
				// Maximums.
				em, _ := funcs.Encode(&max, funcs.MaximumEncodingOrder)
				funcs.LazyAppendFile(maxpath, expiration, em)

				// Edges.
				// Not sure if I even care about Edges anymore.
				key := fmt.Sprintf("%d_%s_%s", max.Timestamp, max.Underlying, max.OptionType)
				if edges[key].OptionAsk <= 0 {
					edges[key] = max
					continue
				}
				// Not ultimately sure how to handle pre-computing edges when accounting for commission.
				// This will vary from broker to broker.. so would become fairly complicated if totally generalized.
				if funcs.Multiplier(max.MaximumBid, max.OptionAsk, 2.2) > funcs.Multiplier(edges[key].MaximumBid, edges[key].OptionAsk, 2.2) {
					edges[key] = max
				}
			}
		}

		// Encode.
		es := []structs.Maximum{}
		for _, e := range edges {
			es = append(es, e)
		}
		encodedEdges, _ := c.SerializeMaximums(es)

		// Save to c.livedir + "/edges/" + timestamp
		path := c.livedir + "/edges"
		filename := expiration
		err := funcs.LazyWriteFile(path, filename, []byte(encodedEdges))
		if err != nil {
			c.logError("maybeCycleMaximums", err)
			return
		}

		// Delete.
		delete(c.maximums, expiration)
	}
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

func (c *Collector) parseLogLine(yyyymmdd string, line string) (utcTimestamp int64, _type string, encodedEquity string) {
	t, _ := time.Parse("20060102", yyyymmdd)
	yyyymmdd_in_seconds := t.Unix()

	columns := strings.Split(line, ",")
	if len(columns) < 3 {
		return -1, "", ""
	}

	hhmmss_in_seconds, err := strconv.ParseInt(columns[0], 10, 64)
	if err != nil {
		c.logError("parseLogLine", err)
		return -1, "", ""
	}

	utcTimestamp = yyyymmdd_in_seconds + hhmmss_in_seconds
	_type = columns[1]
	encodedEquity = strings.Join(columns[2:], ",")

	return utcTimestamp, _type, encodedEquity
}

func (c *Collector) parseQuoteLine(line string) (utcTimestamp int64, _type string, encodedEquity string) {
	columns := strings.Split(line, ",")
	if len(columns) < 3 {
		return -1, "", ""
	}
	utcTimestamp, err := strconv.ParseInt(columns[0], 10, 64)
	if err != nil {
		c.logError("parseLogLine", err)
		return -1, "", ""
	}

	_type = columns[1]
	encodedEquity = strings.Join(columns[2:], ",")

	return utcTimestamp, _type, encodedEquity
}

func (c *Collector) promoteTarget(t target) {
	if t.Stock.Symbol == "" {
		message := fmt.Sprintf("Empty Target, Discarding, t.Timestamp: %d", t.Timestamp)
		c.logError("promoteTarget", message)
		return
	}

	lines, err := encodeTarget(t)
	if err != nil {
		c.logError("promoteTarget", err)
		return
	}
	yyyymmdd := time.Unix(t.Timestamp, 0).Format("20060102")
	funcs.LazyAppendFile(c.livedir+"/quotes", yyyymmdd, lines)

	// touch appropriate /live/timestamp/<ts> filename.
	ts := fmt.Sprintf("%d", t.Timestamp)
	err = funcs.LazyTouchFile(c.livedir+"/timestamp", ts)
	if err != nil {
		c.logError("promoteTarget", err)
	}

	// This is bit that concerns me.. but not sure what other ugly things would need to be done to avoid.
	for _, o := range t.Options {
		c.addMaximum(o, t.Stock, t.Timestamp)
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

func (c *Collector) SerializeMaximums(maximums []structs.Maximum) (string, error) {
	var err error
	encodedMaximums := ""
	for _, maximum := range maximums {
		m, e := funcs.Encode(&maximum, funcs.MaximumEncodingOrder)
		if e != nil {
			fmt.Println("[SerializeMaximums] Err encoding maximum: %s", err)
			err = e
			continue
		}
		encodedMaximums += m + "\n"
	}
	return encodedMaximums[:len(encodedMaximums)-len("\n")], err
}

func (c *Collector) updateMaximum(o structs.Option, timestamp int64) {
	// Don't do anything if nothing has been promoted for this expiration.
	// All initialization happens inside addMaximum()
	_, expExists := c.maximums[o.Expiration]
	if !expExists {
		return
	}
	_, optionSymbolExists := c.maximums[o.Expiration][o.Symbol]
	if !optionSymbolExists {
		return
	}

	for idx, _ := range c.maximums[o.Expiration][o.Symbol] {
		if o.Bid > c.maximums[o.Expiration][o.Symbol][idx].MaximumBid {
			c.maximums[o.Expiration][o.Symbol][idx].MaximumBid = o.Bid
			c.maximums[o.Expiration][o.Symbol][idx].MaxTimestamp = timestamp
		}
	}
}

func (c *Collector) updateTarget(utcTimestamp int64, _type string, encodedEquity string) (o structs.Option, err error) {
	switch _type {
	case "s":
		s := structs.Stock{}
		funcs.Decode(encodedEquity, &s, funcs.StockEncodingOrder)

		err = c.updateStockTarget(s, utcTimestamp)
	case "o":
		funcs.Decode(encodedEquity, &o, funcs.OptionEncodingOrder)

		err = c.updateOptionTarget(o, utcTimestamp)
	default:
		c.logError("updateTarget", "Bad switch _type: "+_type)
	}

	return o, err
}

func (c *Collector) updateOptionTarget(o structs.Option, utc_timestamp int64) error {
	hhmmss_in_seconds := utc_timestamp % int64(24*60*60)
	near, distance := isNear(hhmmss_in_seconds, o.Time, 45)
	if !near {
		return fmt.Errorf("%d seconds is too far away.", distance)
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

	return nil
}

func (c *Collector) updateStockTarget(s structs.Stock, utc_timestamp int64) error {
	hhmmss_in_seconds := utc_timestamp % int64(24*60*60)
	near, distance := isNear(hhmmss_in_seconds, s.Time, 45)
	if !near {
		return fmt.Errorf("%d seconds is too far away. hhmmss_in_seconds: %d", distance, hhmmss_in_seconds)
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

	return nil
}

func encodeTarget(t target) (string, error) {
	es, err := funcs.Encode(&t.Stock, funcs.StockEncodingOrder)
	if err != nil {
		return "", err
	}
	lines := fmt.Sprintf("%d,s,%s", t.Timestamp, es)
	for _, o := range t.Options {
		eo, err := funcs.Encode(&o, funcs.OptionEncodingOrder)
		if err != nil {
			return "", err
		}
		lines += fmt.Sprintf("\n%d,o,%s", t.Timestamp, eo)
	}
	return lines, nil
}

func getEdgeKey(e structs.Maximum) string {
	edgeID := funcs.TimestampID(e.Timestamp)
	return fmt.Sprintf("%s_%s_%d", e.Underlying, e.OptionType, edgeID)
}

func getOptionTypeFromEdgeKey(edgeKey string) string {
	return strings.Split(edgeKey, "_")[1]
}

func getTimestampIDFromEdgeKey(edgeKey string) int64 {
	tsID := strings.Split(edgeKey, "_")[2]
	val, _ := strconv.ParseInt(tsID, 10, 0)
	return val
}

func getUnderlyingFromEdgeKey(edgeKey string) string {
	return strings.Split(edgeKey, "_")[0]
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
