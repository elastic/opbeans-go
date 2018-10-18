package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgin"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/module/apmsql"
)

const (
	cacheURLFormat = "'inmem' or 'redis://user:pass@host'"
)

var (
	listenAddr      = flag.String("listen", ":8000", "Address on which to listen for HTTP requests")
	backendAddrs    = flag.String("backend", "", "Comma-separated list of addresses of opbeans services to proxy API requests to ($OPBEANS_SERVICES)")
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

	apm.DefaultTracer.SetLogger(logger)

	// Instrument the default HTTP transport, so that outgoing
	// (reverse-proxy) requests are reported as spans.
	http.DefaultTransport = apmhttp.WrapRoundTripper(http.DefaultTransport)

	if err := Main(logger); err != nil {
		logger.Fatal(err)
	}
}

func Main(logger *logrus.Logger) error {
	frontendBuildDir := filepath.FromSlash(*frontendDir)
	indexFilePath := filepath.Join(frontendBuildDir, "index.html")
	faviconFilePath := filepath.Join(frontendBuildDir, "favicon.ico")
	staticDirPath := filepath.Join(frontendBuildDir, "static")
	imagesDirPath := filepath.Join(frontendBuildDir, "images")

	var backendURLs []*url.URL
	if *backendAddrs == "" {
		*backendAddrs = os.Getenv("OPBEANS_SERVICES")
	}
	if *backendAddrs != "" {
		for _, field := range strings.Split(*backendAddrs, ",") {
			field = strings.TrimSpace(field)
			if u, err := url.Parse(field); err == nil && u.Scheme != "" {
				backendURLs = append(backendURLs, u)
				continue
			}
			// Not an absolute URL, so should be a host or host/port pair.
			hostport := field
			if _, _, err := net.SplitHostPort(hostport); err != nil {
				// A bare host was specified; assume the same port
				// that we're listening on.
				_, port, err := net.SplitHostPort(*listenAddr)
				if err != nil {
					port = "3000"
				}
				hostport = net.JoinHostPort(hostport, port)
			}
			backendURLs = append(backendURLs, &url.URL{Scheme: "http", Host: hostport})
		}
	}

	db, err := newDatabase(logger)
	if err != nil {
		return err
	}
	defer db.Close()

	cacheStore, err := newCache()
	if err != nil {
		return err
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(apmgin.Middleware(r))
	r.Use(cache.Cache(&cacheStore))

	pprof.Register(r)
	r.Static("/static", staticDirPath)
	r.Static("/images", imagesDirPath)
	r.StaticFile("/favicon.ico", faviconFilePath)
	r.StaticFile("/", indexFilePath)
	r.GET("/oopsie", handleOopsie)
	r.GET("/rum-config.js", handleRUMConfig)
	r.Use(func(c *gin.Context) {
		// Paths used by the frontend for state.
		for _, prefix := range []string{
			"/dashboard",
			"/products",
			"/customers",
			"/orders",
		} {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				c.Request.URL.Path = "/"
				r.HandleContext(c)
				return
			}
		}
		c.Next()
	})

	// Create API routes. We install middleware for /api which probabilistically
	// proxies these requests to another opbeans service to demonstrate distributed
	// tracing, and test agent compatibility.
	proxyProbability := 0.5
	if value := os.Getenv("OPBEANS_DT_PROBABILITY"); value != "" {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse OPBEANS_DT_PROBABILITY")
		}
		if f < 0.0 || f > 1.0 {
			return errors.Errorf("invalid OPBEANS_DT_PROBABILITY value %s: out of range [0,1.0]", value)
		}
		proxyProbability = f
	}
	rand.Seed(time.Now().UnixNano())
	maybeProxy := func(c *gin.Context) {
		if len(backendURLs) > 0 && rand.Float64() < proxyProbability {
			u := backendURLs[rand.Intn(len(backendURLs))]
			logger.Infof("proxying API request to %s", u)
			httputil.NewSingleHostReverseProxy(u).ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Next()
	}
	apiGroup := r.Group("/api", maybeProxy)
	addAPIHandlers(apiGroup, db, logger)

	return r.Run(*listenAddr)
}

func handleRUMConfig(c *gin.Context) {
	apmServerURL := os.Getenv("ELASTIC_APM_JS_SERVER_URL")
	if apmServerURL == "" {
		apmServerURL = "http://localhost:8200"
	} else {
		apmServerURL = template.JSEscapeString(apmServerURL)
	}
	content := fmt.Sprintf("window.elasticApmJsBaseServerUrl = '%s';\n", apmServerURL)
	c.Data(200, "text/javascript", []byte(content))
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

func newDatabase(logger *logrus.Logger) (*sqlx.DB, error) {
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
	dbx := sqlx.NewDb(db, driver)
	if err := initDatabase(dbx, driver, logger); err != nil {
		db.Close()
		return nil, err
	}
	return dbx, nil
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

func handleOopsie(c *gin.Context) {
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
