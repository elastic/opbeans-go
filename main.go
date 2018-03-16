package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmgin"
	"github.com/elastic/apm-agent-go/contrib/apmsql"
	_ "github.com/elastic/apm-agent-go/contrib/apmsql/pq"
	_ "github.com/elastic/apm-agent-go/contrib/apmsql/sqlite3"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	cacheURLFormat = "'inmem' or 'redis://user:pass@host'"
)

var (
	listenAddr = flag.String("listen", ":8000", "Address on which to listen for HTTP requests")
	database   = flag.String("db", "sqlite3:demo/db.sql", "Database URL")
	cacheURL   = flag.String("cache", "inmem", "Cache URL ("+cacheURLFormat+")")
)

func main() {
	flag.Parse()
	logger := logrus.New()
	if err := Main(logger); err != nil {
		logger.Fatal(err)
	}
}

func Main(logger *logrus.Logger) error {
	db, err := newDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	cache, err := newCache()
	if err != nil {
		return err
	}

	elasticapm.DefaultTracer.SetProcessor(apmgin.Processor{})
	elasticapm.DefaultTracer.SetLogger(logrus.StandardLogger())

	r := newRouter(logger, cache)
	pprof.Register(r)
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.Static("/images", "./demo/images")
	r.LoadHTMLGlob("templates/*")
	r.GET("/", handleIndex)
	r.GET("/panic", handlePanic)

	indexPrefixes := []string{"/dashboard", "/products", "/customers", "/orders"}
	for _, path := range []string{"/dashboard", "/products", "/customers", "/orders"} {
		r.GET(path, handleIndex)
	}
	r.NoRoute(func(c *gin.Context) {
		for _, prefix := range indexPrefixes {
			if !strings.HasPrefix(c.Request.URL.Path, prefix+"/") {
				continue
			}
			handleIndex(c)
			return
		}
		c.Next()
	})

	addAPIHandlers(r.Group("/api"), db, logger)
	return r.Run(*listenAddr)
}

func newDatabase() (*sql.DB, error) {
	fields := strings.SplitN(*database, ":", 2)
	if len(fields) != 2 {
		return nil, errors.Errorf(
			"expected database URL with format %q, got %q",
			"<driver>:<connection-string>",
			*database,
		)
	}
	db, err := apmsql.Open(fields[0], fields[1])
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func newCache() (persistence.CacheStore, error) {
	const defaultExpiration = time.Minute
	if *cacheURL == "inmem" {
		return persistence.NewInMemoryStore(defaultExpiration), nil
	}
	if !strings.HasPrefix(*cacheURL, "redis") {
		return nil, errors.Errorf(
			"invalid cache URL %q, expected %s",
			*cacheURL, cacheURLFormat,
		)
	}
	redisPool := newRedisPool(*cacheURL)
	return persistence.NewRedisCacheWithPool(redisPool, defaultExpiration), nil
}

func newRedisPool(url string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(url)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if _, err := c.Do("PING"); err != nil {
				return err
			}
			return nil
		},
	}
}

func newRouter(logger *logrus.Logger, cacheStore persistence.CacheStore) *gin.Engine {
	r := gin.New()
	// TODO(axw) use ginrus when we have configuration
	// for logging to elasticsearch.
	//r.Use(ginrus.Ginrus(logger, time.RFC3339, true))
	//r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(apmgin.Middleware(r, nil))
	r.Use(cache.Cache(&cacheStore))
	return r
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
}

func handlePanic(c *gin.Context) {
	switch c.Query("type") {
	case "string":
		panic("boom")
	case "pkg/errors":
		err := errors.New("boom")
		panic(errors.Wrap(err, "failure while shaking the room"))
	default:
		panic(fmt.Errorf("sonic %s", "boom"))
	}
}
