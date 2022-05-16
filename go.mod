module github.com/xyzj/gopsu

go 1.16

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/denisenkom/go-mssqldb v0.10.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/gzip v0.0.3
	github.com/gin-gonic/gin v1.7.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/goccy/go-json v0.9.7
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pebbe/zmq4 v1.2.7
	github.com/pkg/errors v0.9.1
	github.com/tealeg/xlsx v1.0.5
	github.com/tidwall/gjson v1.9.3
	github.com/tidwall/sjson v1.1.6
	github.com/unrolled/secure v1.0.9
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2 // indirect
	golang.org/x/text v0.3.6
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20220512140231-539c8e751b99
)
