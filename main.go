package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmgin"
	"github.com/elastic/apm-agent-go/module/apmsql"

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
	listenAddr      = flag.String("listen", ":8000", "Address on which to listen for HTTP requests")
	database        = flag.String("db", "sqlite3::memory:", "Database URL")
	frontendDir     = flag.String("frontend", "frontend/build", "Frontend assets dir")
	cacheURL        = flag.String("cache", "inmem", "Cache URL ("+cacheURLFormat+")")
	healthcheckAddr = flag.String("healthcheck", "", "Address to connect to for Docker healthchecking")
)

func main() {
	flag.Parse()
	logger := logrus.StandardLogger()
	if *healthcheckAddr != "" {
		if err := healthcheck(logger); err != nil {
			logger.Errorf("healthcheck failed: %s", err)
			os.Exit(1)
		}
		return
	}

	elasticapm.DefaultTracer.SetLogger(logger)
	if err := Main(logger); err != nil {
		logger.Fatal(err)
	}
}

func Main(logger *logrus.Logger) error {
	db, err := newDatabase(logger)
	if err != nil {
		return err
	}
	defer db.Close()

	cache, err := newCache()
	if err != nil {
		return err
	}

	frontendBuildDir := filepath.FromSlash(*frontendDir)
	indexFilePath := filepath.Join(frontendBuildDir, "index.html")
	faviconFilePath := filepath.Join(frontendBuildDir, "favicon.ico")
	staticDirPath := filepath.Join(frontendBuildDir, "static")
	imagesDirPath := filepath.Join(frontendBuildDir, "images")

	indexContent, err := prepareIndex(indexFilePath)
	if err != nil {
		return err
	}
	handleIndex := func(c *gin.Context) {
		c.Data(200, "text/html", indexContent)
	}

	r := newRouter(cache)
	pprof.Register(r)
	r.Static("/static", staticDirPath)
	r.Static("/images", imagesDirPath)
	r.StaticFile("/favicon.ico", faviconFilePath)
	r.GET("/", handleIndex)
	r.GET("/panic", handlePanic)

	indexPrefixes := []string{"/dashboard", "/products", "/customers", "/orders"}
	for _, path := range indexPrefixes {
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

// prepareIndex reads index.html, and injects a javascript
// tag to configure the RUM agent.
func prepareIndex(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, errors.Errorf("%q not found: make sure you build frontend first", path)
	} else if err != nil {
		return nil, err
	}

	tag := []byte("<body>")
	i := bytes.Index(content, tag)
	if i == -1 {
		return content, nil
	}

	apmServerURL := os.Getenv("ELASTIC_APM_JS_SERVER_URL")
	if apmServerURL == "" {
		apmServerURL = "http://localhost:8200"
	} else {
		apmServerURL = template.JSEscapeString(apmServerURL)
	}

	var buf bytes.Buffer
	buf.Write(content[:i+len(tag)])
	buf.WriteString(fmt.Sprintf(`
<script type="application/javascript">
	window.ElasticApmJsBaseServiceName = 'opbeans-web'
	window.ElasticApmJsBaseServerUrl = '%s'
</script>
`, apmServerURL))
	buf.Write(content[i+len(tag):])
	return buf.Bytes(), nil
}

func healthcheck(logger *logrus.Logger) error {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/orders", *healthcheckAddr))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var orders []Order
	return json.NewDecoder(resp.Body).Decode(&orders)
}

func newDatabase(logger *logrus.Logger) (*sql.DB, error) {
	fields := strings.SplitN(*database, ":", 2)
	if len(fields) != 2 {
		return nil, errors.Errorf(
			"expected database URL with format %q, got %q",
			"<driver>:<connection-string>",
			*database,
		)
	}
	driver := fields[0]
	db, err := apmsql.Open(driver, fields[1])
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := initDatabase(db, driver, logger); err != nil {
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

func newRouter(cacheStore persistence.CacheStore) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(apmgin.Middleware(r))
	r.Use(cache.Cache(&cacheStore))
	return r
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
