module istio.io/pkg

go 1.12

replace github.com/golang/glog => github.com/istio/glog v0.0.0-20190424172949-d7cfb6fa2ccd

replace github.com/spf13/viper => github.com/istio/viper v1.3.3-0.20190515210538-2789fed3109c

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-multierror v1.0.0
	github.com/howeyc/fsnotify v0.9.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v0.9.4
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/prometheus/prom2json v1.1.0
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.3.0
	go.opencensus.io v0.20.2
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.20.1
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.2.7
	gotest.tools v2.2.0+incompatible
	istio.io/api v0.0.0-20190515205759-982e5c3888c6
)
