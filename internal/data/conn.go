package data

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/shared/v1"

	_ "github.com/go-sql-driver/mysql"

	"github.com/google/uuid"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type (
	query interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}

	Conn struct {
		query
		generateUUID uuidgen
		logger       *log.Entry
	}

	uuidgen func() shared.UUID

	getMockDB func() *sql.DB
)

var mtrcs = metrics.DataMetrics.MustCurryWith(prometheus.Labels{"pkg": "mysql"})

func NewInstance(dbuser, dbpass, dbhost string, dbport uint16, logger *log.Entry) (*Conn, error) {
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
	return &Conn{db, uuidGen, l}, nil
}

func trackError(cid shared.CID, l *log.Entry, m *prometheus.CounterVec, err error, lvs ...string) error {
	m.WithLabelValues(lvs...).Inc()
	l.WithError(err).WithField("CID", cid).Error("???")
	return err
}

func uuidGen() shared.UUID {
	return shared.UUID(uuid.New().String())
}

// or should i make a conn_test.go just for these?
func mockUUIDGen() shared.UUID {
	return shared.UUID(uuid.Must(uuid.FromBytes([]byte("0123456789abcdef"))).String())
}

func testLogger(_ *testing.T, fields log.Fields) *log.Entry {
	return (&log.Logger{
		Out:       os.Stderr, // testWriter(t),
		Hooks:     make(log.LevelHooks),
		Formatter: &log.JSONFormatter{},
		Level:     log.DebugLevel,
	}).
		WithFields(fields)
}
