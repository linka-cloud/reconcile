module go.linka.cloud/reconcile

go 1.15

require (
	github.com/golang/protobuf v1.4.3
	github.com/stretchr/testify v1.7.0
	go.linka.cloud/libkv/store/boltdb/v2 v2.0.0
	go.linka.cloud/libkv/v2 v2.0.0
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	google.golang.org/protobuf v1.25.0
	k8s.io/apimachinery v0.20.2 // indirect
	k8s.io/client-go v11.0.0+incompatible
)

replace (
	go.linka.cloud/libkv/store/boltdb/v2 => github.com/linka-cloud/libkv/store/boltdb/v2 v2.0.0-20210129144330-02d275120b73
	go.linka.cloud/libkv/v2 => github.com/linka-cloud/libkv/v2 v2.0.0
)
