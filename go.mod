module go.linka.cloud/reconcile

go 1.15

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	go.linka.cloud/libkv/store/boltdb/v2 v2.0.0
	go.linka.cloud/libkv/v2 v2.0.0
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace (
	go.linka.cloud/libkv/store/boltdb/v2 => github.com/linka-cloud/libkv/store/boltdb/v2 v2.0.0-20210129144330-02d275120b73
	go.linka.cloud/libkv/v2 => github.com/linka-cloud/libkv/v2 v2.0.0
)
