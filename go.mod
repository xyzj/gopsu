module github.com/xyzj/gopsu

go 1.16

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/coreos/bbolt v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/denisenkom/go-mssqldb v0.10.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/gzip v0.0.3
	github.com/gin-gonic/gin v1.7.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/google/uuid v1.2.0
	github.com/json-iterator/go v1.1.11
	github.com/pebbe/zmq4 v1.2.7
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0 // indirect
	github.com/streadway/amqp v1.0.0
	github.com/tidwall/gjson v1.7.5
	github.com/tidwall/sjson v1.1.6
	github.com/unrolled/secure v1.0.9
	go.etcd.io/etcd v3.3.25+incompatible
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/text v0.3.6
	google.golang.org/grpc v1.26.0
)
