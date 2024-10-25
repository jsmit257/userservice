package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jsmit257/userservice/internal/data"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/internal/router"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const APP_NAME = "serve-mysql"

var traps = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGHUP,
	syscall.SIGQUIT}

func main() {
	cfg := NewConfig()

	// TODO:
	//   - os.MkdirAll(cfg.LogDir)
	//   - logFileBasePath := cfg.LogDir + os.PathSeparator + cfg.LogFileName
	//   - logFilePath := logFileBasePath + os.Process.Pid
	//   - var logWriter *io.Writer
	//   - if logFile, err := os.OpenFile(logFilePath, os.O_CREATE+os.???, 644); err != nil {
	//   - 	panic(fmt.Errorf("can't create logfile: '%s'", logFilePath))
	//   - } else {
	//   - 	logWriter = bufio.NewWriter(logFile)
	//   - 	defer logFile.Close()
	//   - }
	//   - os.Link(logFilePath, logFileBasePath)
	//   - log.SetOutput(logWriter)	log.SetOutput(os.Stderr)
	// TODO: replace magic constant DebugLevel below
	// logLevel, err := log.ParseLevel("LogLevel")
	// if err != nil {
	// 	// logLevel = log.DebugLvel  // fix it?
	// }
	// log.SetLevel(logLevel)
	log.SetLevel(log.DebugLevel)
	// log.SetReportCaller(true) // this seems expensive, maybe nice to have, check it out before enabling
	log.SetFormatter(&log.JSONFormatter{})

	logger := log.WithField("app", APP_NAME)

	logger.Debug("loaded config")

	mtrcs := metrics.NewHandler(prometheus.NewRegistry())

	data, err := data.NewInstance(
		cfg.MySQLUser,
		cfg.MySQLPwd,
		cfg.MySQLHost,
		cfg.MySQLPort,
		logger)
	cfg.MySQLPwd = "*****" // kinda rude
	if err != nil {
		logger.
			WithFields(log.Fields{"cfg": cfg}).
			WithError(err).
			Fatal("failed to connect mysql client")
		return
	}

	logger.Debug("configured mysql client")

	srv := router.NewInstance(
		&router.UserService{
			Userer:    data,
			Addresser: data,
			Contacter: data,
		},
		cfg.ServerHost,
		cfg.ServerPort,
		mtrcs,
		logger)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(srv *http.Server, wg *sync.WaitGroup) {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.
				WithFields(log.Fields{"cfg": cfg}).
				WithError(err).
				Fatal("http server didn't start properly")
			panic(err)
		}
		logger.Debug("http server shutdown complete")
	}(srv, wg)

	trap(logger)

	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(timeout); err != nil {
		logger.
			WithFields(log.Fields{"cfg": cfg}).
			WithError(err).
			Error("error stopping server")
	}

	wg.Wait()

	logger.Debug("done")
}

func trap(logger *log.Entry) {
	trapped := make(chan os.Signal, len(traps))

	signal.Notify(trapped, traps...)

	logger.WithField("sig", <-trapped).Info("stopping app with signal")
}
