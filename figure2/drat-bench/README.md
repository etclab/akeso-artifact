### Cost of Key Rotation w/ Double Ratchet

- inside directory `drat-bench`
- make: \
```make all```
- test one: \
```./key-manager 4```
- run benchmark: \
```go test ./internal/key-manager -bench=. -benchmem > result.out```
- plot: \
```gnuplot -c plots/drat-bench.gpi > plots/drat-bench-plot.eps && epstopdf plots/drat-bench-plot.eps```
