package valid

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var (
	ctx = context.Background()
	cid = shared.CID("Test_Validator")
	cfg = &config.Config{
		AuthnTimeout: 15,
		CookieName:   "foobar",
		MaxLogins:    5,
	}
	expireme = time.Duration(cfg.AuthnTimeout*60) * time.Second
)

func Test_Login(t *testing.T) {
	t.Parallel()

	logins := []string{"1", "2", "3", "4", "5", "6"}

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Login")
	v := NewValidator(db, cfg, l)

	// // count fails
	mock.ExpectSMembers("logins:userid").SetErr(fmt.Errorf("some error"))
	valid, sc := v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// failed exists token
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// too many members
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	for _, v := range logins {
		mock.ExpectExists(v).SetVal(1)
	}
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusTooManyRequests, sc)
	require.Nil(t, valid)

	// did cleanup but still too many members
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	for _, v := range logins[1:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:userid", "1").SetVal(1)
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusTooManyRequests, sc)
	require.Nil(t, valid)

	// error removing tokens
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	mock.ExpectExists("2").SetVal(0)
	for _, v := range logins[2:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:userid", "1", "2").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// failed to set token
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// failed to set expiry
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// couldn't find the token we just created - ???
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(false)
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// add to index fails
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Nil(t, valid)

	// finally! the happy login path
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	mock.ExpectExists("2").SetVal(0)
	for _, v := range logins[2:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:userid", "1", "2").SetVal(2)
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	valid, sc = v.Login(ctx, userid, remote, cid)
	require.Equal(t, http.StatusOK, sc)
	require.NotNil(t, valid)
	require.NotEmpty(t, valid.Value)
}

func Test_Valid(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Valid")
	v := NewValidator(db, cfg, l)

	// exists fails
	mock.ExpectExists("token:token").SetErr(fmt.Errorf("some error"))
	_, sc := v.Valid(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, sc)

	// doesn't exist
	mock.ExpectExists("token:token").SetVal(0)
	_, sc = v.Valid(ctx, "token", cid)
	require.Equal(t, http.StatusForbidden, sc)

	// update expiry fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetErr(fmt.Errorf("some error"))
	_, sc = v.Valid(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, sc)

	// token doesn't exist (any more? how?)
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(false)
	_, sc = v.Valid(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, sc)

	// happy path for valid
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	_, sc = v.Valid(ctx, "token", cid)
	require.Equal(t, http.StatusFound, sc)
}

func Test_Logout(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Logout")
	v := NewValidator(db, cfg, l)

	// valid gets an error
	mock.ExpectExists("token:token").SetVal(0)
	cookie, code := v.Logout(ctx, "token", cid)
	require.Equal(t, http.StatusForbidden, code, cid)
	require.Nil(t, cookie)

	// get userid fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// delete hash fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// remove from index fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetVal(1)
	mock.Regexp().ExpectSRem("logins:.*", "token:.*").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token", cid)
	require.Equal(t, http.StatusInternalServerError, code, cid)
	require.Nil(t, cookie)

	// happy logout
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetVal(1)
	mock.Regexp().ExpectSRem("logins:.*", "token:.*").SetVal(1)
	cookie, code = v.Logout(ctx, "token", cid)
	require.Equal(t, http.StatusAccepted, code, cid)
	require.Equal(t, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	}, cookie)
}
