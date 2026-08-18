module github.com/nofendian17/sbterm/apps/ws

go 1.26.5

replace github.com/nofendian17/sbterm/libs/proto => ../../libs/proto

replace github.com/nofendian17/sbterm/libs/pkg => ../../libs/pkg

require (
	github.com/gorilla/websocket v1.5.3
	github.com/nofendian17/sbterm/libs/pkg v0.0.0-00010101000000-000000000000
	github.com/nofendian17/sbterm/libs/proto v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.12.0
	github.com/twmb/franz-go v1.21.6
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.13.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
