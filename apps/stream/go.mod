module github.com/nofendian17/sbterm/apps/stream

go 1.26.5

replace github.com/nofendian17/sbterm/libs/proto => ../../libs/proto

replace github.com/nofendian17/sbterm/libs/pkg => ../../libs/pkg

replace github.com/nofendian17/sbterm/libs/marketdata => ../../libs/marketdata

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/gorilla/websocket v1.5.3
	github.com/nofendian17/sbterm/libs/marketdata v0.0.0-00010101000000-000000000000
	github.com/nofendian17/sbterm/libs/pkg v0.0.0-00010101000000-000000000000
	github.com/nofendian17/sbterm/libs/proto v0.0.0-00010101000000-000000000000
	github.com/samber/do/v2 v2.1.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.12.0
	github.com/twmb/franz-go v1.21.6
	github.com/twmb/franz-go/pkg/kfake v0.0.0-20260816150254-beb096adff00
	github.com/twmb/franz-go/pkg/kmsg v1.13.1
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
