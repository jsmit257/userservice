package data

import (
	"context"
	"database/sql"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/internal/router"
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
		sqls         config.Sqls
		logger       *log.Entry
	}

	uuidgen func() shared.UUID
)

var mtrcs = metrics.DataMetrics.MustCurryWith(prometheus.Labels{
	"pkg": "data",
	"app": "userservice",
	"db":  "mysql",
})

func NewConn(db *sql.DB, sqls config.Sqls, l *log.Entry) *router.UserService {
	conn := &Conn{db, uuidGen, sqls, l}
	return &router.UserService{
		Addresser: conn,
		Contacter: conn,
		Userer:    conn,
	}
}

// nolint: unused
func trackError(cid shared.CID, l *log.Entry, m *prometheus.CounterVec, err error, lvs ...string) error {
	m.WithLabelValues(lvs...).Inc()
	l.WithError(err).WithField("CID", cid).Error("???")
	return err
}

func uuidGen() shared.UUID {
	return shared.UUID(uuid.NewString())
}

func hash(pass, salt string) string {
	return pass + salt
}

func generateSalt() string {
	return "salt"
}
