ERI's Phake
-----------
Phake is a simple tool to test ERI. In it's most basic form you use it to train ERI and use a load-testing tool (like Vegeta) to test it's performance

Typical use-scenario:
```sh
$ go run . -num-addr 500000 -gen-domain=true > targets.json
$ cat targets.json | vegeta attack -format=json -rate=500 -duration 5m | tee result.bin | vegeta report
Requests      [total, rate, throughput]  150000, 500.00, 500.00
Duration      [total, attack, wait]      5m0.001624521s, 4m59.99805229s, 3.572231ms
Latencies     [mean, 50, 95, 99, max]    2.57993ms, 695.632µs, 7.853531ms, 8.548955ms, 42.965982ms
Bytes In      [total, mean]              8003430, 53.36
Bytes Out     [total, mean]              15076830, 100.51
Success       [ratio]                    100.00%
Status Codes  [code:count]               200:150000
Error Set:
```

To test cache-miss scenarios:
1. Clear ERI's known list (at moment of writing, just restart ERI)
1. Generate a new set of targets to train ERI with and
1. Use the previous targets.json to attack with