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

	"github.com/jsmit257/userservice/internal/config"
	data "github.com/jsmit257/userservice/internal/relational"
	"github.com/jsmit257/userservice/internal/router"

	log "github.com/sirupsen/logrus"
)

const APP_NAME = "serve-mysql"

var traps = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGHUP,
	syscall.SIGQUIT}

func main() {
	cfg := config.NewConfig()

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

	db, err := newInstance(
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

	sqls, err := config.NewSqls("mysql")
	if err != nil {
		logger.
			WithFields(log.Fields{"cfg": cfg}).
			WithError(err).
			Fatal("failed to connect mysql client")
	}

	us := data.NewConn(db, sqls, logger)

	srv := router.NewInstance(
		us,
		cfg,
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

func newInstance(dbuser, dbpass, dbhost string, dbport uint16, logger *log.Entry) (*sql.DB, error) {
	l := logger.WithFields(log.Fields{
		"mysql_user":     dbuser,
		"mysql_hostname": dbhost,
		"mysql_port":     dbport,
	})
	l.Debug("starting mysql conn")
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/userservice?parseTime=true", dbuser, dbpass, dbhost, dbport)
	db, err := sql.Open("mysql", url)
	if err != nil {
		l.WithError(err).Error("failed to create mysql conn")
		return nil, err
	} else if err = db.Ping(); err != nil {
		l.WithError(err).Error("failed to ping mysql conn")
		return nil, err
	}
	l.Info("successfully connected to mysql")
	return db, nil
}
