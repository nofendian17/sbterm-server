module github.com/nofendian17/sbterm/libs/marketdata

go 1.26.5

replace github.com/nofendian17/sbterm/libs/proto => ../proto

require (
	github.com/nofendian17/sbterm/libs/proto v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.12.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
