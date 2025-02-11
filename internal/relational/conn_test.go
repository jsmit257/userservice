package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	row    []string
	values []driver.Value
	repl   struct {
		ndx uint
		val any
	}

	getMockDB func(*sql.DB, sqlmock.Sqlmock, error) *sql.DB
)

var (
	rightaboutnow = time.Now().UTC()
	testmetrics   = metrics.DataMetrics.MustCurryWith(prometheus.Labels{
		"pkg": "data test",
	})
)

func Test_NewUserService(t *testing.T) {
	result := NewUserService(nil, nil, logrus.WithTime(time.Now().UTC()), testmetrics)
	require.NotNil(t, result)
}

func (v values) nil(ord ...uint) values {
	for _, o := range ord {
		v = v.replace(repl{o, nil})
	}
	return v
}

func (v values) replace(r ...repl) values {
	result := make(values, len(v))
	copy(result, v)

	for _, sub := range r {
		result[sub.ndx] = sub.val
	}

	return result
}

// func Test_hash(t *testing.T) {
// 	require.Equal(t, 64, len(hash("foobar", generateSalt())), hash("foobar", generateSalt()))
// }

func mockUUIDGen() shared.UUID {
	return shared.UUID(uuid.Must(uuid.FromBytes([]byte("0123456789abcdef"))).String())
}

func mockSqls() config.Sqls {
	result := make(config.Sqls, 4)
	for _, table := range []string{"address", "basic-auth", "contact", "user"} {
		temp := make(map[string]string, 5)
		for _, verb := range []string{"select-all", "select", "insert", "update", "delete"} {
			temp[verb] = "snakeoil"
		}
		result[table] = temp
	}
	return result
}

func mockContext(cid shared.CID) context.Context {
	return context.WithValue(
		context.WithValue(
			context.WithValue(
				context.Background(),
				shared.CTXKey("log"),
				logrus.WithField("app", "test"),
			),
			shared.CTXKey("metrics"),
			metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{
				"proto":  "test",
				"method": "test",
				"url":    "test",
			}),
		),
		shared.CTXKey("cid"),
		cid,
	)
}

func testLogger(_ *testing.T, fields logrus.Fields) *logrus.Entry {
	return (&logrus.Logger{
		Out:       os.Stderr, // testWriter(t),
		Hooks:     make(logrus.LevelHooks),
		Formatter: &logrus.JSONFormatter{},
		Level:     logrus.DebugLevel,
	}).
		WithFields(fields)
}
