module istio.io/pkg

go 1.15

replace github.com/golang/glog => github.com/istio/glog v0.0.0-20190424172949-d7cfb6fa2ccd

replace github.com/spf13/viper => github.com/istio/viper v1.3.3-0.20190515210538-2789fed3109c

replace golang.org/x/tools => golang.org/x/tools v0.0.0-20191216173652-a0e659d51361

require (
	cloud.google.com/go v0.76.0 // indirect
	cloud.google.com/go/logging v1.2.0
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-multierror v1.0.0
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v0.9.4
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/prometheus/prom2json v1.1.0
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.6.1
	go.opencensus.io v0.22.6
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/oauth2 v0.0.0-20210201163806-010130855d6c // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	google.golang.org/genproto v0.0.0-20210203152818-3206188e46ba
	google.golang.org/grpc v1.35.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/klog/v2 v2.4.0
)
