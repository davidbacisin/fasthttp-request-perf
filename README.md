# fasthttp HTTP Client Performance Tests
Performance tests for using [fasthttp](https://github.com/valyala/fasthttp) as a 
client rather than a server

Code examples and a discussion of my benchmarking results can be found on 
[my blog](https://davidbacisin.com/writing/using-fasthttp-for-api-requests-golang).

# Running the benchmarks
You'll need Go 1.13+ to support Go modules.

Run the benchmarks using Go's built-in testing and benchmarking tools:

```
go test -bench='OverTCP' -benchmem -benchtime=10s
go test -bench='MockServer' -benchmem -benchtime=10s
```

Because fasthttp is built to minimize memory allocations, I've included the 
`-benchmem` flag to also measure memory usage.