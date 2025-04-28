package valid

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var (
	ctx = context.WithValue(
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
		shared.CID("test"),
	)

	cfg = &config.Config{
		AuthnTimeout: 15,
		CookieName:   "foobar",
		MaxLogins:    5,
	}
	expireme = time.Duration(cfg.AuthnTimeout*60) * time.Second
)

func Test_Login(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Login")
	v := NewValidator(db, cfg, l)
	logoutCookie := v.(*core).logoutCookie

	ctx := setcid("count fails, any reason")
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	valid, sc := v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	ctx = setcid("failed to set token")
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	ctx = setcid("failed to set expiry")
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	ctx = setcid("couldn't find the token we just created - ???")
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(false)
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	ctx = setcid("add to index fails")
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	ctx = setcid("finally! the happy login path")
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusOK, sc)
	require.NotNil(t, valid)
	require.NotEmpty(t, valid.Value)
}

func Test_Valid(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Valid")
	v := NewValidator(db, cfg, l)

	ctx := setcid("exists fails")
	mock.ExpectExists("token:token").SetErr(fmt.Errorf("some error"))
	_, sc := v.Valid(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("doesn't exist")
	mock.ExpectExists("token:token").SetVal(0)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusTemporaryRedirect, sc)

	ctx = setcid("update expiry fails")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetErr(fmt.Errorf("some error"))
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("token doesn't exist (any more? how?)")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(false)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusTemporaryRedirect, sc)

	ctx = setcid("happy path for valid")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusNoContent, sc)
}

func Test_Logout(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Logout")
	v := NewValidator(db, cfg, l)

	ctx := setcid("valid gets an error")
	mock.ExpectExists("token:token").SetVal(0)
	cookie, code := v.Logout(ctx, "token")
	require.Equal(t, http.StatusTemporaryRedirect, code)
	require.Nil(t, cookie)

	ctx = setcid("get userid fails")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	ctx = setcid("get userid nil")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetErr(redis.Nil)
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusForbidden, code)
	require.Nil(t, cookie)

	ctx = setcid("delete hash fails")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	ctx = setcid("remove from index fails")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetVal(1)
	mock.Regexp().ExpectSRem("logins:.*", "token:.*").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	ctx = setcid("happy logout")
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetVal(1)
	mock.Regexp().ExpectSRem("logins:.*", "token:.*").SetVal(1)
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusNoContent, code)
	require.Equal(t, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	}, cookie)
}

func Test_OTP(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_OTP")
	v := NewValidator(db, cfg, l)

	ctx := setcid("count fails, any reason")
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	pad, sc := v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, pad)

	ctx = setcid("happy path")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "pad:.*").SetVal(1)
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusOK, sc, pad)
	require.NotEmpty(t, pad)

	ctx = setcid("err creating hash")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetErr(fmt.Errorf("some error"))
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)

	ctx = setcid("err expiring new hash")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetErr(fmt.Errorf("some error"))
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)

	ctx = setcid("error adding pad to logins")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "pad:.*").SetErr(fmt.Errorf("some error"))
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)
	require.NotEmpty(t, pad)

	ctx = setcid("adding pad to login returns 0")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "pad:.*").SetVal(0)
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)
	require.NotEmpty(t, pad)
}

func Test_LoginOTP(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_LoginOTP")
	v := NewValidator(db, cfg, l)

	ctx := setcid("fails finding a pad")
	mock.ExpectHGetAll("pad:1").SetErr(fmt.Errorf("some error"))
	loc, sc := v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, "/", loc)

	ctx = setcid("missing userid")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{})
	loc, sc = v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, "/", loc)

	ctx = setcid("empty userid")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: ""})
	loc, sc = v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusBadRequest, sc)
	require.Equal(t, "/", loc)

	ctx = setcid("redirect is missing")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: userid})
	loc, sc = v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, "/", loc)

	ctx = setcid("redirect is empty")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		redirect: "",
	})
	loc, sc = v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusBadRequest, sc)
	require.Equal(t, "/", loc)

	ctx = setcid("all happy")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		redirect: redirect,
	})
	loc, sc = v.LoginOTP(ctx, "1")
	require.Equal(t, http.StatusFound, sc)
	require.Equal(t, redirect, loc)
}

func Test_CompleteOTP(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_CompleteOTP")
	v := NewValidator(db, cfg, l)

	ctx := setcid("fails finding a pad")
	mock.ExpectHGetAll("pad:1").SetErr(fmt.Errorf("some error"))
	id, sc := v.CompleteOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, shared.UUID(""), id)

	ctx = setcid("missing userid")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{})
	id, sc = v.CompleteOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, shared.UUID(""), id)

	ctx = setcid("empty userid")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: ""})
	id, sc = v.CompleteOTP(ctx, "1")
	require.Equal(t, http.StatusBadRequest, sc)
	require.Equal(t, shared.UUID(""), id)

	ctx = setcid("clear logins fails (any reason)")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: userid})
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	id, sc = v.CompleteOTP(ctx, "1")
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, shared.UUID(""), id)

	ctx = setcid("happy path")
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: userid})
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	id, sc = v.CompleteOTP(ctx, "1")
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, shared.UUID(userid), id)
}

func Test_clearLogins(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_CompleteOTP")
	v := NewValidator(db, cfg, l).(*core)

	ctx = setcid("fails getting logins for cleartokens")
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	sc := v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("happy path with no logins (redis.Nil)")
	mock.ExpectSMembers("logins:" + userid).SetErr(redis.Nil)
	sc = v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusGone, sc)

	ctx = setcid("happy path with no logins (empty list)")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{})
	sc = v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusGone, sc)

	ctx = setcid("failed to vacuum a login")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetErr(fmt.Errorf("some error"))
	sc = v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("can't remove token from logins")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetVal(3)
	mock.ExpectSRem("logins:"+userid, "pad:1").SetErr(fmt.Errorf("some error"))
	sc = v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("happy path")
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetVal(3)
	mock.ExpectSRem("logins:"+userid, "pad:1").SetVal(1)
	sc = v.clearLogins(ctx, userid)
	require.Equal(t, http.StatusGone, sc)
}

func Test_checkCount(t *testing.T) {
	t.Parallel()

	logins := []string{"1", "2", "3", "4", "5", "6"}

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_checkCount")
	v := NewValidator(db, cfg, l).(*core)

	ctx := setcid("count fails")
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	sc := v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("failed exists token")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins)
	mock.ExpectExists("1").SetErr(fmt.Errorf("some error"))
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("too many members")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins)
	for _, v := range logins {
		mock.ExpectExists(v).SetVal(1)
	}
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusTooManyRequests, sc)

	ctx = setcid("happy path (nothing to remove)")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins[:2])
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusOK, sc)

	ctx = setcid("error removing tokens")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	mock.ExpectExists("2").SetVal(0)
	for _, v := range logins[2:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:"+userid, "1", "2").SetErr(fmt.Errorf("some error"))
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusInternalServerError, sc)

	ctx = setcid("did cleanup but still too many members")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	for _, v := range logins[1:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:"+userid, "1").SetVal(1)
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusTooManyRequests, sc)

	ctx = setcid("happy path (after remove)")
	mock.ExpectSMembers("logins:" + userid).SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	mock.ExpectExists("2").SetVal(0)
	for _, v := range logins[2:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:"+userid, "1", "2").SetVal(2)
	sc = v.checkCount(ctx, "logins:"+userid)
	require.Equal(t, http.StatusOK, sc)
}

func Test_trackererror(t *testing.T) {
	defer func() {
		require.NotNil(t, recover())
	}()
	t.Parallel()

	db, _ := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_CheckOTP")
	v := NewValidator(db, cfg, l)

	tracker := v.(*core).tracker(ctx, "test_trackerror")
	tracker = tracker.err(NotAuthorized)
	tracker.err(NotAuthorized)
}

func setcid(val string) context.Context {
	return context.WithValue(ctx, shared.CTXKey("cid"), shared.CID(val))
}
