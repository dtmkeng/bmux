## Test Router and param Aero and Hybrid AeroXMux
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