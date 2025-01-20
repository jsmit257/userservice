package data

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/maild"
	"github.com/jsmit257/userservice/internal/router"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	Conn struct {
		query
		uuidgen
		sqls    config.Sqls
		mx      maild.Sender
		log     *logrus.Entry
		metrics *prometheus.CounterVec
	}

	query interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}

	uuidgen func() shared.UUID

	deferred func(err error, l *logrus.Entry) error

	userVec struct{ *prometheus.CounterVec }
)

func NewUserService(db *sql.DB, sqls config.Sqls, mx maild.Sender, l *logrus.Entry, m *prometheus.CounterVec) *router.UserService {
	conn := &Conn{db, uuidGen, sqls, mx, l, m}
	return &router.UserService{
		Addresser: conn,
		Auther:    conn,
		Contacter: conn,
		Userer:    conn,
	}
}

func (db *Conn) logging(fn string, key any, cid shared.CID) (deferred, *logrus.Entry) {
	start := time.Now()

	l := db.log.WithFields(logrus.Fields{
		"method": fn,
		"cid":    cid,
	})

	if key != nil {
		l = l.WithField("key", key)
	}

	m := userVec{db.metrics}.labels(prometheus.Labels{"function": fn})

	l.Info("starting work")

	return func(err error, l *logrus.Entry) error {
		if err != nil {
			l = l.WithError(err)
		}
		l.WithField("duration", time.Since(start).String()).Infof("finished work")
		m.done(err)
		return err
	}, l
}

func (u userVec) labels(l prometheus.Labels) userVec {
	u.CounterVec = u.CounterVec.MustCurryWith(l)
	return u
}

func (u userVec) done(err error) {
	if err == nil {
		u.WithLabelValues("none").Inc()
	} else {
		u.WithLabelValues(err.Error()).Inc()
	}
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
