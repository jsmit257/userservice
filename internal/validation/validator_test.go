package valid

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
)

func Test_Validator(t *testing.T) {
	// t.Skip()
	t.Parallel()

	ctx := context.Background()
	cid := shared.CID("Test_Validator")
	cfg := &config.Config{
		AuthnTimeout: 15,
		CookieName:   "foobar",
		// RedisHost:    "localhost",
		// RedisPort:    6666,
	}

	db, mock := redismock.NewClientMock()

	v := NewValidator(db, cfg)
	require.NotNil(t, v)

	// increment fails
	mock.Regexp().ExpectIncr("user:.*").SetErr(fmt.Errorf("some error"))
	valid, sc := v.Login(ctx, "userid", "remote", cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// too many logged in users
	mock.Regexp().ExpectIncr("user:.*").SetVal(6)
	mock.Regexp().ExpectDecr("user:.*").SetVal(5)
	valid, sc = v.Login(ctx, "userid", "remote", cid)
	require.Equal(t, http.StatusTooManyRequests, sc)
	require.Nil(t, valid)

	// decrement fails
	mock.Regexp().ExpectIncr("user:.*").SetVal(6)
	mock.Regexp().ExpectDecr("user:.*").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, "userid", "remote", cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// set new value fails
	mock.Regexp().ExpectIncr("user:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		"userid":  "userid",
		"remote":  "remote",
		"expires": ".*",
	}).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, "userid", "remote", cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// finally! the happy login path
	mock.Regexp().ExpectIncr("user:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		"userid":  "userid",
		"remote":  "remote",
		"expires": ".*",
	}).SetVal(1)
	valid, sc = v.Login(ctx, "userid", "remote", cid)
	require.Equal(t, http.StatusOK, sc)

	// valid can't get expiry
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal("snakeoil")
	require.Equal(t, http.StatusInternalServerError, v.Valid(ctx, valid.Value, cid))

	// expiry is too old
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339))
	require.Equal(t, http.StatusForbidden, v.Valid(ctx, valid.Value, cid))

	// token returns err
	mock.ExpectHGet("token:"+valid.Value, "expires").SetErr(fmt.Errorf("some error"))
	require.Equal(t, http.StatusInternalServerError, v.Valid(ctx, valid.Value, cid))

	// token returns nil
	mock.ExpectHGet("token:"+valid.Value, "expires").RedisNil()
	require.Equal(t, http.StatusForbidden, v.Valid(ctx, valid.Value, cid))

	// happy path for valid
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Format(time.RFC3339))
	require.Equal(t, http.StatusFound, v.Valid(ctx, valid.Value, cid))

	// valid gets an error
	mock.ExpectHGet("token:"+valid.Value, "expires").SetErr(fmt.Errorf("some error"))
	cookie, code := v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// get userid fails
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Format(time.RFC3339))
	mock.ExpectHGet("token:"+valid.Value, "userid").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// delete hash fails
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Format(time.RFC3339))
	mock.ExpectHGet("token:"+valid.Value, "userid").SetVal("12345")
	mock.ExpectHDel("token:"+valid.Value, "userid", "expires", "remote").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// decrement fails
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Format(time.RFC3339))
	mock.ExpectHGet("token:"+valid.Value, "userid").SetVal("12345")
	mock.ExpectHDel("token:"+valid.Value, "userid", "expires", "remote").SetVal(1)
	mock.Regexp().ExpectDecr("user:*").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// happy logout
	mock.ExpectHGet("token:"+valid.Value, "expires").SetVal(time.Now().UTC().Format(time.RFC3339))
	mock.ExpectHGet("token:"+valid.Value, "userid").SetVal("12345")
	mock.ExpectHDel("token:"+valid.Value, "userid", "expires", "remote").SetVal(1)
	mock.Regexp().ExpectDecr("user:*").SetVal(0)
	cookie, code = v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusAccepted, code, cid)
	require.Equal(t, logoutCookie(cfg.CookieName), cookie)
}
