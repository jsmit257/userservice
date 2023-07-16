package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jsmit257/userservice/internal/data/mysql"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/internal/router"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var traps = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGHUP,
	syscall.SIGQUIT}

func main() {
	cfg := NewConfig()

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	// log.SetReportCaller(true) // this seems expensive, maybe nice to have, check it out before enabling
	log.SetFormatter(&log.JSONFormatter{})

	l := log.WithField("cfg", cfg)

	mtrcs := metrics.NewHandler(prometheus.NewRegistry())

	mysql, err := mysql.NewInstance(
		cfg.MySQLUser,
		cfg.MySQLRootPwd,
		cfg.MySQLHost,
		cfg.MySQLPort)
	if err != nil {
		l.WithError(err).Fatal("failed to start client")
	}

	l.Debug("configured userservice")

	srv := router.NewInstance(
		&router.UserService{
			User:    mysql,
			Address: mysql,
			Contact: mysql,
		},
		cfg.ServerHost,
		cfg.ServerPort,
		mtrcs)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(srv *http.Server, wg *sync.WaitGroup) {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			l.WithError(err).Fatal("http server didn't start properly")
		}
		l.Debug("http server shutdown complete")
	}(srv, wg)

	trap()

	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(timeout); err != nil {
		l.WithError(err).Error("error stopping server")
	}

	wg.Wait()

	l.Debug("done")
}

func trap() {
	trapped := make(chan os.Signal, len(traps))

	signal.Notify(trapped, traps...)

	log.WithField("sig", <-trapped).Info("stopping app with signal") // FIXME:
}
