package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_Collector_collect(t *testing.T) {
	c := New("test", "./testdir", int64(60))

	// Suppose may need some sort of LoadAdapterFromConfig() function somewhere.
	lines, _ := funcs.GetConfig(c.rootdir + "/config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]
	jsess := lines[3]

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		lines[3] = tda.JsessionID
		funcs.UpdateConfig(c.rootdir+"/config", lines)
	}
	c.Adapter = tda
	symbol := "INTC"
	thisMonth, limitMonth := c.collect(symbol)

	select {
	case reply := <-c.replies:
		r, _ := reply.(bool)
		if !r {
			t.Errorf("Received False Reply from c.replies")
		}
	case reply := <-c.pipe:
		s, success := reply.Data.(structs.Stock)
		if !success {
			t.Errorf("Expected to get Stock back first.")
		}
		if s.Symbol != symbol {
			t.Errorf("Expected: %s, Got: %s!", symbol, s.Symbol)
		}
	}

	if len(c.pipe) == 0 {
		t.Errorf("Expecting Non-zero Pipe.")
	}

	seenThisMonth := false
	seenLimitMonth := false
	// If here, should read out all options.
	for message := range c.pipe {
		if message.Data == nil {
			break
		}
		o, success := message.Data.(structs.Option)
		if !success {
			t.Errorf("Expecting an Option here!")
		}
		if o.Expiration[:6] != thisMonth && o.Expiration[:6] != limitMonth {
			t.Errorf("Expected Expiration %s or %s, Got: %s", thisMonth, limitMonth, o.Expiration)
		}
		if o.Expiration[:6] == thisMonth {
			seenThisMonth = true
		}
		if o.Expiration[:6] == limitMonth {
			seenLimitMonth = true
		}
	}

	if !seenThisMonth {
		t.Errorf("Did not see Exp: %s", thisMonth)
	}
	if !seenLimitMonth {
		t.Errorf("Did not see Exp: %s", limitMonth)
	}
}

func Test_Collector_dumpTargets(t *testing.T) {
	c := New("test", "./testdir", int64(60))

	c.targets = map[string]map[string]target{"current": map[string]target{}, "next": map[string]target{}}
	c.targets["current"]["AAPL"] = target{Timestamp: int64(1234567890)}
	c.targets["current"]["BABA"] = target{Timestamp: int64(1234567890)}

	c.dumpTargets()
}

func Test_Collector_GetPastNEdges(t *testing.T) {
	c := New("test", "./testdir", int64(60))

	timestamp := int64(1421257800)
	n := 4
	edges := c.GetPastNEdges(timestamp, n)
	expirations := map[string]bool{}

	for _, edge := range edges {
		if edge.Expiration == "" {
			continue
		}
		expirations[edge.Expiration] = true
	}

	if len(expirations) != n {
		t.Errorf("Expected %d Expirations! Got: %v", n, len(expirations))
	}

	if len(edges)%n != 0 {
		t.Errorf("Expected len(edges) to be multiple of %d.  Got: %d!", n, len(edges))
	}
}

func Test_Collector_GetQuotes(t *testing.T) {
	// Munge off of collectord data.
	c := New("test", "../cmd/collectord/testdir", int64(60))
	t1, _ := time.Parse("20060102 15:04 MST", "20150123 12:00 EST")
	utcTimestamp := t1.UTC().Unix()

	quotes, err := c.GetQuotes(utcTimestamp, "AAPL")
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(quotes) == 0 {
		t.Errorf("Expected more than 0 quotes.")
	}
	quotes, err = c.GetQuotes(utcTimestamp+int64(10*60), "AAPL")
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(quotes) == 0 {
		t.Errorf("Expected more than 0 quotes.")
	}
}

func Test_Collector_loadTargets(t *testing.T) {
	c := New("test", "./testdir", int64(60))

	targets := c.loadTargets()

	if targets == nil {
		t.Errorf("Expected, at least, empty targets!")
	}
	if len(targets["current"]) != 2 {
		t.Errorf("Expecting 2 'current' targets.")
	}
	if len(targets["next"]) > 0 {
		t.Errorf("Not expecting 'next' targets.")
	}

	os.RemoveAll(c.livedir + "/targets")
}

func Test_Collector_isNear(t *testing.T) {
	padding := 45

	// EST Testing.
	time1 := funcs.ClockTimeInSeconds("204801")
	time2 := funcs.ClockTimeInSeconds("154801")
	near, diff := isNear(time1, time2, padding)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 0 {
		t.Errorf("Expected 0, Got: %d", int(diff))
	}

	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("154851")
	near, diff = isNear(time1, time2, padding)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}

	// EDT Testing
	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("164821")
	near, diff = isNear(time1, time2, padding)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 20 {
		t.Errorf("Expected 20, Got: %d", int(diff))
	}

	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("164851")
	near, diff = isNear(time1, time2, padding)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}
}

func Test_Collector_logError(t *testing.T) {
	c := New("test", "./testdir", int64(60))
	os.RemoveAll(c.errordir)
	c.logError("testfunc", fmt.Errorf("Test error of type 'error'"))
	c.logError("testfunc", "Test error of type 'string'")
}

func Test_Collector_maybeCycleMaximums(t *testing.T) {
	c := New("test", "./testdir", int64(60))
	c.maybeCycleMaximums(int64(1000))
}

func Test_Collector_maybeCycleTargets(t *testing.T) {
	c := New("test", "./testdir", int64(60))
	start_ts := int64(10 * 60)
	next_ts := start_ts + int64(10*60)
	c.targets["current"]["GOOG"] = target{Timestamp: start_ts}
	c.targets["next"]["GOOG"] = target{Timestamp: next_ts}

	// Should silently pass over old timestamp.
	ts := c.targets["current"]["GOOG"].Timestamp - 50
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != start_ts {
		t.Errorf("Current Target Timestamp should not have advanced!")
	}
	if c.targets["next"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Next Target Timestamp should not have advanced!")
	}

	// Should cycle for timestamp past 5 minute midpoint.
	ts = c.targets["current"]["GOOG"].Timestamp + int64(5*60)
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Target Timestamp Expected: %d, Got: %d", next_ts, c.targets["current"]["GOOG"].Timestamp)
	}
	if c.targets["next"]["GOOG"].Timestamp == next_ts {
		t.Errorf("Next Target Timestamp should have advanced!")
	}
}

// Mainly, wish to verify maximums is updated.
func Test_Collector_promoteTarget(tst *testing.T) {
	c := New("test", "./testdir", int64(60))
	os.RemoveAll(c.livedir + "/maximums")

	exp := "20150130"
	symbol := "GOOG_013015C600"
	t1, _ := time.Parse("20060102 15:04", "20150126 19:00")
	ts := t1.Unix()

	t := target{}
	t.Timestamp = ts
	t.Stock.Symbol = "GOOG"
	t.Stock.Bid = 600

	t.Options = map[string]structs.Option{}
	t.Options[symbol] = structs.Option{Symbol: symbol, Bid: 200, Ask: 300, Volume: 100, Strike: 60000, Underlying: "GOOG", Expiration: exp}

	c.promoteTarget(t)

	if len(c.maximums[exp][symbol]) != 1 {
		tst.Errorf("Expected to find my option!")
	}
}

func Test_Collector_SerializeMaximums_DeserializeMaximums(t *testing.T) {
	c := New("test", "./testdir", int64(60))
	maximums := []structs.Maximum{
		structs.Maximum{Underlying: "GOOG", OptionSymbol: "GOOG_013015C600"},
		structs.Maximum{Underlying: "AAPL", OptionSymbol: "AAPL_013015P100"}}

	sm, err := c.SerializeMaximums(maximums)
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
	dms, err := c.DeserializeMaximums(sm)
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
	if !reflect.DeepEqual(maximums, dms) {
		t.Errorf("Expected:\n%v\nGot:\n%v", maximums, dms)
	}

}

func Test_Collector_updateTarget(t *testing.T) {
	c := New("test", "./testdir", int64(60))

	t1, _ := time.Parse("20060102 15:04", "20150101 21:00")
	utcTimestamp := t1.Unix()
	encodedEquity := "GOOG,GOOG_011715P655,20150117,57600,65500,15000,15410,7220,0,1,0.00000,p"
	o, err := c.updateTarget(utcTimestamp, "o", encodedEquity)

	if err != nil {
		t.Errorf("Got err: %s", err)
	}

	if !reflect.DeepEqual(c.targets["current"][o.Underlying].Options[o.Symbol], o) {
		t.Errorf("Expected: %v, Got: %v", o, c.targets["current"][o.Underlying].Options[o.Symbol])
	}

	if c.targets["current"][o.Underlying].Timestamp != utcTimestamp {
		t.Errorf("Expected: %v, Got: %v", utcTimestamp, c.targets["current"][o.Underlying].Timestamp)
	}
}

// Kitchen sinking this since don't want to do over and over.
func Test_Collector_addMaximum_updateMaximum_dumpMaximums_loadMaximums(t *testing.T) {
	c := New("test", "./testdir", int64(60))
	exp := "20150130"
	symbol := "GOOG_013015C600"
	t1, _ := time.Parse("20060102 15:04", "20150123 21:00")
	ts := t1.Unix()

	if len(c.maximums) != 0 {
		t.Errorf("Expected 0 maximums, Got: %d", len(c.maximums))
	}

	s := structs.Stock{Symbol: "GOOG", Bid: 60000}
	o := structs.Option{Symbol: symbol, Bid: 500, Ask: 600, Volume: 100, Strike: 60000, Underlying: "GOOG", Expiration: exp}

	err := c.addMaximum(o, s, ts)
	if err == nil {
		t.Errorf("Expected err.")
	}
	if len(c.maximums) != 0 {
		t.Errorf("Expected 0 maximums, Got: %d", len(c.maximums))
	}

	err = c.addMaximum(o, s, ts+int64(2*24*60*60))

	if err != nil {
		t.Errorf("Did not expect error. Got: %s", err)
	}
	if c.maximums[o.Expiration][o.Symbol][0].MaximumBid != o.Bid {
		t.Errorf("Expected MaximumBid to equal option.Bid.")
	}

	o.Bid -= 10
	c.updateMaximum(o, ts)

	if c.maximums[o.Expiration][o.Symbol][0].MaximumBid == o.Bid {
		t.Errorf("Did not expect MaximumBid to change.")
	}

	o.Bid += 100
	c.updateMaximum(o, ts)

	if c.maximums[o.Expiration][o.Symbol][0].MaximumBid != o.Bid {
		t.Errorf("Expected: %d, Got: %d", o.Bid, c.maximums[o.Expiration][o.Symbol][0].MaximumBid)
	}

	maximums := c.maximums

	c.dumpMaximums()

	c.maximums = map[string]map[string][]structs.Maximum{}

	if reflect.DeepEqual(c.maximums, maximums) {
		t.Errorf("Expected mismatched maximums!")
	}

	c.maximums = c.loadMaximums()

	if !reflect.DeepEqual(c.maximums, maximums) {
		t.Errorf("Expected %v, Got: %v", maximums, c.maximums)
	}
}
