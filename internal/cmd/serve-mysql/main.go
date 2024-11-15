package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	data "github.com/jsmit257/userservice/internal/relational"
	"github.com/jsmit257/userservice/internal/router"
	valid "github.com/jsmit257/userservice/internal/validation"
)

const APP_NAME = "serve-mysql"

var traps = []os.Signal{
	os.Interrupt,
	syscall.SIGHUP,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

func main() {
	var err error
	cfg := config.NewConfig()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log := logrus.WithField("app", APP_NAME)

	log.Info("loaded config and configured logger")

	defer cleanup(log, cfg, err)

	db, metrics, err := newMysql(cfg)
	cfg.MySQLPwd = "*****" // kinda rude
	if err != nil {
		panic("failed to connect mysql client")
	}
	log.Info("configured mysql client")

	sqls, err := config.NewSqls("mysql")
	if err != nil {
		panic("failed to connect mysql client")
	}
	log.Info("fetched mysql DML")

	authn, err := newRedis(cfg)
	if err != nil {
		panic("failed to connect redis client")
	}
	log.Info("created redis authn store")

	us := data.NewUserService(db, sqls, log, metrics)
	us.Validator = valid.NewValidator(authn, cfg, log)

	srv := router.NewInstance(us, cfg, log)

	startServer(srv, log).Wait()

	log.Debug("done")
}

func cleanup(log *logrus.Entry, cfg *config.Config, err error) {
	l := log.WithFields(logrus.Fields{"cfg": cfg}).WithError(err)

	msg := recover()
	if msg == nil {
		return
	}

	switch text := msg.(type) {
	case string:
		l.Fatal(text)
	case error:
		l.Fatal(text.Error())
	case interface{ String() string }:
		l.Fatal(text.String())
	default:
		l.Fatalf("%v", msg)
	}
}

func newMysql(cfg *config.Config) (*sql.DB, *prometheus.CounterVec, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/userservice?parseTime=true",
		cfg.MySQLUser,
		cfg.MySQLPwd,
		cfg.MySQLHost,
		cfg.MySQLPort,
	)
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, nil, err
	} else if err = db.Ping(); err != nil {
		return nil, nil, err
	}
	return db, metrics.DataMetrics.MustCurryWith(prometheus.Labels{
		"app": APP_NAME,
		"db":  "mysql",
		"pkg": "data",
	}), nil
}

func newRedis(cfg *config.Config) (*redis.Client, error) {
	url := fmt.Sprintf("redis://%s:%s@%s:%d/0",
		cfg.RedisUser,
		cfg.RedisPass,
		cfg.RedisHost,
		cfg.RedisPort)
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	_, err = client.Ping(context.Background()).Result()

	return client, err
}

func startServer(srv *http.Server, log *logrus.Entry) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()

	go func(srv *http.Server) {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Fatal("http server didn't start properly")
		}
	}(srv)
	log.Info("server started, waiting for traps")

	trapped := make(chan os.Signal, len(traps))

	signal.Notify(trapped, traps...)

	log.WithField("sig", <-trapped).Info("stopping app with signal")

	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(timeout); err != nil {
		log.WithError(err).Error("error stopping server")
	}

	log.Debug("http server shutdown complete")

	return wg
}
