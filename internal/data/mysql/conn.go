package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jsmit257/userservice/internal/metrics"

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
	}

	uuidgen func() uuid.UUID

	getMockDB func() *sql.DB
)

var mtrcs = metrics.DataMetrics.MustCurryWith(prometheus.Labels{"pkg": "mysql"})

func NewInstance(dbuser, dbpass, dbhost string, dbport uint16) (*Conn, error) {
	l := log.WithFields(log.Fields{
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
	return &Conn{db, uuid.New}, nil
}

func mockUUIDGen() uuid.UUID {
	return uuid.Must(uuid.FromBytes([]byte("0123456789abcdef")))
}
