module github.com/nofendian17/sbterm/apps/ingest

go 1.26.5

replace github.com/nofendian17/sbterm/libs/proto => ../../libs/proto

replace github.com/nofendian17/sbterm/libs/pkg => ../../libs/pkg

replace github.com/nofendian17/sbterm/libs/marketdata => ../../libs/marketdata

require (
	github.com/nofendian17/sbterm/libs/marketdata v0.0.0-00010101000000-000000000000
	github.com/nofendian17/sbterm/libs/pkg v0.0.0-00010101000000-000000000000
	github.com/nofendian17/sbterm/libs/proto v0.0.0-00010101000000-000000000000
	github.com/questdb/go-questdb-client/v4 v4.2.1-0.20260730155217-4f2723e2d5cb
	github.com/samber/do/v2 v2.1.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.12.0
	github.com/twmb/franz-go v1.21.6
	github.com/twmb/franz-go/pkg/kfake v0.0.0-20260816150254-beb096adff00
	github.com/twmb/franz-go/pkg/kmsg v1.13.1
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v28.3.3+incompatible // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.3.3 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241223144023-3abc09e42ca8 // indirect
	google.golang.org/grpc v1.67.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
