package config

import (
	"fmt"
	"os"

	"github.com/beego/beego/v2/core/logs"
	"golang.org/x/net/http2"
)

type AppConfig struct {
	loggR *logs.BeeLogger
}

func (cnf *AppConfig) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", os.Getenv("APP.HOST"), os.Getenv("APP.PORT"))
}

func (cnf *AppConfig) GetRedisURL() string {
	return os.Getenv("REDIS.URL")
}

func (cnf *AppConfig) Http2() *http2.Server {
	return &http2.Server{}
}

func (cnf *AppConfig) GetEnvironment() string {
	switch {
	case cnf.IsTesting():
		return "testing"
	case cnf.IsProduction():
		return "production"
	default: // Default to development
		return "development"
	}
}

func (cnf *AppConfig) IsTesting() bool {
	return os.Getenv("APP.ENV") == "testing"
}

func (cnf *AppConfig) IsDevelopment() bool {
	return os.Getenv("APP.ENV") == "development"
}

func (cnf *AppConfig) IsProduction() bool {
	return os.Getenv("APP.ENV") == "production"
}

func (cnf *AppConfig) Logger() *logs.BeeLogger {
	beeLogger := logs.NewLogger(10000)

	beeLogger.SetLogger(
		logs.AdapterConsole,
		`{"level":6,"color":true}`,
	)

	beeLogger.EnableFuncCallDepth(true)
	beeLogger.SetLogFuncCallDepth(3)

	beeLogger.Async(1000)

	cnf.loggR = beeLogger

	return beeLogger
}

func New() *AppConfig {
	return &AppConfig{}
}
