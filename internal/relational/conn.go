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
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	Conn struct {
		query
		uuidgen
		sqls config.Sqls
		// mx      maild.Sender
		log     *logrus.Entry
		metrics *prometheus.CounterVec
	}

	query interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}

	uuidgen func() shared.UUID

	deferred func(err error, l *logrus.Entry) error

	userVec struct{ *prometheus.CounterVec }
)

func NewUserService(db *sql.DB, sqls config.Sqls, l *logrus.Entry, m *prometheus.CounterVec) *Conn {
	return &Conn{db, uuidGen, sqls, l.WithFields(logrus.Fields{
		"pkg": "data",
		"db":  "mysql",
	}), m.MustCurryWith(prometheus.Labels{
		"db": "mysql",
	})}
}

// func Obfuscate(s string) string {
// 	result, _ := uuid.FromBytes([]byte(s + "abcdefghijklmnop")[:16])
// 	sum := sha256.Sum256([]byte(result.String()))
// 	return hex.EncodeToString(sum[:])
// }

func (db *Conn) logging(fn string, key any, cid shared.CID) (deferred, *logrus.Entry) {
	start := time.Now()

	l := db.log.WithFields(logrus.Fields{
		"method": fn,
		"cid":    cid,
	})

	if key != nil {
		l = l.WithFields(logrus.Fields{"key": key})
	}

	m := userVec{db.metrics}.labels(prometheus.Labels{"function": fn})

	l.Info("starting work")

	return func(err error, l *logrus.Entry) error {
		if err != nil {
			l = l.WithError(err)
		}
		l.WithField("duration", time.Since(start).String()).Info("finished work")
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

func hash(pass shared.Password, salt string) shared.Password {
	// result := string(pass)
	// for i, sum := 0, [32]byte{}; i < 483; i++ {
	// 	result += salt
	// 	sum = sha256.Sum256([]byte(result))
	// 	result = string(sum[:])
	// }
	// return shared.Password(hex.EncodeToString([]byte(result)))
	return pass + shared.Password(salt)
}

func generateSalt() string {
	// result := make([]byte, 4)
	// _, _ = rand.Read(result)
	// return string(result)
	return "salt"
}
