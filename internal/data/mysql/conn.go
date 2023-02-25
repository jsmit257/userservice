package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jsmit257/userservice/internal/metrics"

	_ "github.com/go-sql-driver/mysql"

	"github.com/google/uuid"

	"github.com/prometheus/client_golang/prometheus"
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

func NewInstance(dbhost, dbuser, dbpass string, dbport int) (*Conn, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/userservice", dbuser, dbpass, dbhost, dbport)
	db, err := sql.Open("mysql", url)
	if err != nil {
		panic(err)
	} else if err = db.Ping(); err != nil {
		panic(err)
	}
	return &Conn{db, uuid.New}, nil
}

func mockUUIDGen() uuid.UUID {
	return uuid.Must(uuid.FromBytes([]byte("0123456789abcdef")))
}
