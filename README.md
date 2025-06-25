Vague Schematic
===============

![Vague Schematic](https://docs.google.com/drawings/d/101-7Rp9DE7aJXBeks2XlcRYwwUHiWA6PHXaim5Iz6iQ/pub?w=1356&h=335)


Quick Start
===========

Clone and test The Box. 
```
$ cd ~
$ git clone git@github.com:eliwjones/thebox.git
$ cd thebox
$ go test -v ./...
=== RUN   Test_Simulate_Adapter
--- PASS: Test_Simulate_Adapter (0.00s)
=== RUN   Test_Simulate_New
--- PASS: Test_Simulate_New (0.00s)
=== RUN   Test_Simulate_ClosePosition
--- PASS: Test_Simulate_ClosePosition (0.00s)
=== RUN   Test_Simulate_Connect

   ...

--- PASS: Test_Multiplier (0.00s)
=== RUN   Test_NextFriday
--- PASS: Test_NextFriday (0.00s)
=== RUN   Test_updateConfig
--- PASS: Test_updateConfig (0.00s)
=== RUN   Test_WeekID
--- PASS: Test_WeekID (0.00s)
PASS
ok      github.com/eliwjones/thebox/util/funcs  0.003s
```

Historical Overview
===================

NOTE: TD Ameritrade API endpoints were shut down permanently on May 10, 2024.  TODO: build a Schwab Adapter. 

```
a := tdameritrade.New(id, pass, sid, jsessionid)
t := trader.New(a)

b, _ := t.GetBalances()

// Money needs concept of how often to push out Allotments to dispatcher "allotment" channel.
//   also will need to serialize, deserialize that state.
m := money.New(b["value"], b["cash"])

data_dir := "/thebox/datadir"
d := destiny.New(maxageOfPath, data_dir)

// Connect m, d, t to their dispatchers.
m.Subscribe("allotment", "destiny", d.amIn, false)  // destiny wants allotments from money.
t.Subscribe("delta", "destiny", d.dIn, false)  // destiny wants deltas from trader.
t.Subscribe("delta", "money", m.deltaIn, false)  // money wants deltas from trader.
d.Subscribe("protoorder", "trader", t.pomIn, false)  // trader wants protoorders from destiny.

// Send time.Now().UTC() to necessary components, followed by 'shutdown'
//   OR create Pulsar, and simulate (would require using simulate.New() in place of tdameritrade).
p := pulsar.New(data_dir + "/timestamps/now","0000000000","9999999999")
p.Subscribe("money", m.pulses, m.pulsarReply)
p.Subscribe("destiny", d.pulses, d.pulsarReply)
p.Subscribe("trader", t.pulses, t.pulsarReply)

p.Start()

// System will exit once pulsar is done.
// For simulation, simulate adapter will need to subscribe to pulses as well.
```