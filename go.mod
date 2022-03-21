module github.com/elastic/opbeans-go

go 1.14

require (
	github.com/gin-contrib/cache v1.1.0
	github.com/gin-contrib/pprof v0.0.0-20181223171755-ea03ef73484d
	github.com/gin-gonic/gin v1.7.7
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/jmoiron/sqlx v1.3.4
	github.com/lib/pq v1.10.4
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.1
	go.elastic.co/apm/module/apmgin/v2 v2.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/module/apmlogrus/v2 v2.0.0
	go.elastic.co/apm/module/apmsql/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace (
	go.elastic.co/apm/module/apmgin/v2 => github.com/elastic/apm-agent-go/module/apmgin/v2 v2.0.0-20220125052152-dbce0fc5646c
	go.elastic.co/apm/module/apmhttp/v2 => github.com/elastic/apm-agent-go/module/apmhttp/v2 v2.0.0-20220125052152-dbce0fc5646c
	go.elastic.co/apm/module/apmlogrus/v2 => github.com/elastic/apm-agent-go/module/apmlogrus/v2 v2.0.0-20220125052152-dbce0fc5646c
	go.elastic.co/apm/module/apmsql/v2 => github.com/elastic/apm-agent-go/module/apmsql/v2 v2.0.0-20220125052152-dbce0fc5646c
	go.elastic.co/apm/v2 => github.com/elastic/apm-agent-go/v2 v2.0.0-20220125052152-dbce0fc5646c
)
