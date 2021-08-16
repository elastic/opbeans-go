module github.com/elastic/opbeans-go

go 1.14

require (
	github.com/gin-contrib/cache v1.1.0
	github.com/gin-contrib/pprof v0.0.0-20181223171755-ea03ef73484d
	github.com/gin-gonic/gin v1.7.4
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/jmoiron/sqlx v1.3.4
	github.com/lib/pq v1.10.2
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm v1.13.1
	go.elastic.co/apm/module/apmgin v1.13.1
	go.elastic.co/apm/module/apmhttp v1.13.1
	go.elastic.co/apm/module/apmlogrus v1.13.1
	go.elastic.co/apm/module/apmsql v1.13.1
)
