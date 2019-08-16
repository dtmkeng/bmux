## Test Router and param Aero and Hybrid AeroXMux
### Test Default Benchmark
``` s
goos: linux
goarch: amd64
pkg: github.com/dtmkeng/bmux
BenchmarkAero_Param      	20000000	        79.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param      	20000000	        79.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_Param5     	20000000	        61.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param5     	20000000	        63.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_Param20    	20000000	        61.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param20    	20000000	        61.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParamWrite 	20000000	        79.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_ParamWrite 	20000000	        86.6 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dtmkeng/bmux	12.085s

```
### Test With Gtihub API
``` s
   Aero_github: 923304 Bytes
   Bmux_github: 688480 Bytes
goos: linux
goarch: amd64
pkg: github.com/dtmkeng/bmux
BenchmarkAero_Param        	20000000	        78.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param        	20000000	        86.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_Param5       	30000000	        63.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param5       	20000000	        63.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_Param20      	30000000	        60.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_Param20      	30000000	        61.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParamWrite   	20000000	        79.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_ParamWrite   	20000000	        78.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubStatic 	20000000	        90.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_GithubStatic 	20000000	        90.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubParam  	10000000	       208 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_GithubParam  	10000000	       209 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubAll    	   50000	     31567 ns/op	       0 B/op	       0 allocs/op
BenchmarkBmux_GithubAll    	   50000	     31815 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dtmkeng/bmux	26.245s
```