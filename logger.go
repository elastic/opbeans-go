package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.elastic.co/apm/module/apmlogrus"
)

type logLevelFlag struct {
	logrus.Level
}

func (f *logLevelFlag) Set(s string) error {
	level, err := logrus.ParseLevel(s)
	if err != nil {
		return err
	}
	f.Level = level
	return nil
}

func contextLogger(c *gin.Context) logrus.FieldLogger {
	return logrus.WithFields(apmlogrus.TraceContext(c.Request.Context()))
}

func logrusMiddleware(c *gin.Context) {
	start := time.Now()
	method := c.Request.Method
	path := c.Request.URL.Path
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		path += "?" + rawQuery
	}
	c.Next()

	logger := contextLogger(c)
	status := c.Writer.Status()
	entry := logger.WithFields(logrus.Fields{
		"path":      path,
		"method":    method,
		"duration":  time.Since(start),
		"client-ip": c.ClientIP(),
		"status":    status,
	})
	entry.Time = start

	logf := entry.Infof
	switch {
	case status >= http.StatusInternalServerError:
		logf = entry.Errorf
	case status >= http.StatusBadRequest:
		logf = entry.Warningf
	}
	logf("%s %s (%d)", c.Request.Method, c.Request.URL.Path, status)
}

func newJSONFormatter() *logrus.JSONFormatter {
	return &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			// ECS
			logrus.FieldKeyTime:  "@timestamp",
			logrus.FieldKeyLevel: "log.level",
			logrus.FieldKeyMsg:   "message",
			// Non-ECS
			logrus.FieldKeyFunc: "function.name",
		},
	}
}
