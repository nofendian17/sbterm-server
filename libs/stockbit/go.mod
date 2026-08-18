module github.com/nofendian17/sbterm/libs/stockbit

go 1.26.5

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/nofendian17/sbterm/libs/pkg v0.0.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gojek/heimdall/v8 v8.0.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/nofendian17/sbterm/libs/pkg => ../pkg
