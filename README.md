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
=== RUN Test_Simulate_Adapter
--- PASS: Test_Simulate_Adapter (0.00 seconds)
=== RUN Test_Simulate_New
--- PASS: Test_Simulate_New (0.00 seconds)
=== RUN Test_Simulate_Connect
--- PASS: Test_Simulate_Connect (0.00 seconds)
=== RUN Test_Simulate_Get
--- PASS: Test_Simulate_Get (0.00 seconds)
=== RUN Test_Simulate_GetOrders
--- PASS: Test_Simulate_GetOrders (0.00 seconds)
=== RUN Test_Simulate_GetPositions
--- PASS: Test_Simulate_GetPositions (0.00 seconds)
=== RUN Test_Simulate_SubmitOrder
--- PASS: Test_Simulate_SubmitOrder (0.00 seconds)
PASS
ok  	github.com/eliwjones/thebox/adapter/simulate	0.002s
=== RUN Test_Destiny_Get
--- PASS: Test_Destiny_Get (0.00 seconds)
=== RUN Test_Destiny_Put
--- PASS: Test_Destiny_Put (0.00 seconds)
=== RUN Test_Destiny_Decay
--- PASS: Test_Destiny_Decay (0.00 seconds)
=== RUN Test_Destiny_New
--- PASS: Test_Destiny_New (0.00 seconds)
PASS
ok  	github.com/eliwjones/thebox/destiny	0.001s
=== RUN Test_Dispatcher_New
--- PASS: Test_Dispatcher_New (0.00 seconds)
=== RUN Test_Dispatcher_Subscribe
--- PASS: Test_Dispatcher_Subscribe (0.00 seconds)
=== RUN Test_Dispatcher_Allotment
--- PASS: Test_Dispatcher_Allotment (0.00 seconds)
=== RUN Test_Dispatcher_Delta
--- PASS: Test_Dispatcher_Delta (0.00 seconds)
PASS
ok  	github.com/eliwjones/thebox/dispatcher	0.003s
=== RUN Test_Money_New
--- PASS: Test_Money_New (0.00 seconds)
=== RUN Test_Money_Get
--- PASS: Test_Money_Get (0.00 seconds)
=== RUN Test_Money_Put
--- PASS: Test_Money_Put (0.00 seconds)
=== RUN Test_Money_ReAllot
--- PASS: Test_Money_ReAllot (0.00 seconds)
PASS
ok  	github.com/eliwjones/thebox/money	0.002s
=== RUN Test_Trader_New
--- PASS: Test_Trader_New (0.00 seconds)
=== RUN Test_Trader_constructOrder_Option
--- PASS: Test_Trader_constructOrder_Option (0.00 seconds)
=== RUN Test_Trader_constructOrder_Stock
--- PASS: Test_Trader_constructOrder_Stock (0.00 seconds)
PASS
ok  	github.com/eliwjones/thebox/trader	0.002s

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