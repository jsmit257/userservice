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

	logins := []string{"1", "2", "3", "4", "5", "6"}

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Login")
	v := NewValidator(db, cfg, l)
	logoutCookie := v.(*core).logoutCookie

	// // count fails
	mock.ExpectSMembers("logins:userid").SetErr(fmt.Errorf("some error"))
	valid, sc := v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// failed exists token
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// too many members
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	for _, v := range logins {
		mock.ExpectExists(v).SetVal(1)
	}
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusTooManyRequests, sc)
	require.Equal(t, logoutCookie, valid)

	// did cleanup but still too many members
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	for _, v := range logins[1:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:userid", "1").SetVal(1)
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusTooManyRequests, sc)
	require.Equal(t, logoutCookie, valid)

	// error removing tokens
	mock.ExpectSMembers("logins:userid").SetVal(logins)
	mock.ExpectExists("1").SetVal(0)
	mock.ExpectExists("2").SetVal(0)
	for _, v := range logins[2:] {
		mock.ExpectExists(v).SetVal(1)
	}
	mock.ExpectSRem("logins:userid", "1", "2").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// failed to set token
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// failed to set expiry
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// couldn't find the token we just created - ???
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(false)
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

	// add to index fails
	mock.ExpectSMembers("logins:userid").SetVal(logins[:2])
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetErr(fmt.Errorf("some error"))
	valid, sc = v.Login(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, logoutCookie, valid)

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

	// exists fails
	mock.ExpectExists("token:token").SetErr(fmt.Errorf("some error"))
	_, sc := v.Valid(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, sc)

	// doesn't exist
	mock.ExpectExists("token:token").SetVal(0)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusForbidden, sc)

	// update expiry fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetErr(fmt.Errorf("some error"))
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, sc)

	// token doesn't exist (any more? how?)
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(false)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, sc)

	// happy path for valid
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	_, sc = v.Valid(ctx, "token")
	require.Equal(t, http.StatusFound, sc)
}

func Test_Logout(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_Logout")
	v := NewValidator(db, cfg, l)

	// valid gets an error
	mock.ExpectExists("token:token").SetVal(0)
	cookie, code := v.Logout(ctx, "token")
	require.Equal(t, http.StatusForbidden, code)
	require.Nil(t, cookie)

	// get userid fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	// get userid nil
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetErr(redis.Nil)
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusForbidden, code)
	require.Nil(t, cookie)

	// delete hash fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	// remove from index fails
	mock.ExpectExists("token:token").SetVal(1)
	mock.ExpectExpire("token:token", expireme).SetVal(true)
	mock.ExpectHGet("token:token", userid).SetVal("12345")
	mock.ExpectHDel("token:token", userid, remote).SetVal(1)
	mock.Regexp().ExpectSRem("logins:.*", "token:.*").SetErr(fmt.Errorf("some error"))
	cookie, code = v.Logout(ctx, "token")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Nil(t, cookie)

	// happy logout
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

	// happy path
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "pad:.*").SetVal(1)
	pad, sc := v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusOK, sc, pad)
	require.NotEmpty(t, pad)

	// err creating hash
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetErr(fmt.Errorf("some error"))
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)

	// err expiring new hash
	mock.Regexp().ExpectHSet("pad:.*", map[string]interface{}{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("pad:.*", expireme).SetErr(fmt.Errorf("some error"))
	pad, sc = v.OTP(ctx, userid, remote, redirect)
	require.Equal(t, http.StatusInternalServerError, sc, pad)

	// error adding pad to logins
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

	// adding pad to login returns 0
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

	// fails finding a pad
	mock.ExpectHGetAll("pad:1").SetErr(fmt.Errorf("some error"))
	loc, cookie, sc := v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusForbidden, sc, cookie)
	require.Equal(t, "/", loc)

	// missing userid
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{})
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusForbidden, sc, cookie)
	require.Equal(t, "/", loc)

	// empty userid
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{userid: ""})
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusForbidden, sc, cookie)
	require.Equal(t, "/", loc)

	// fails getting logins
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:" + userid).SetErr(fmt.Errorf("some error"))
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusForbidden, sc, cookie)
	require.Equal(t, "/", loc)

	// happy but with no logins to remove (redis.Nil)
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(1)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusFound, sc, cookie)
	require.Equal(t, redirect, loc)

	// happy but with no logins to remove (empty list)
	mock.Regexp().ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	})
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(1)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusFound, sc, cookie)
	require.Equal(t, redirect, loc)

	// can't delete a hash
	mock.Regexp().ExpectHGetAll("pad:.*").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetErr(fmt.Errorf("some error"))
	loc, cookie, sc = v.LoginOTP(ctx, userid, remote)
	require.Equal(t, http.StatusInternalServerError, sc, cookie)
	require.Equal(t, "/", loc)

	// can't remove an element from logins
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetVal(3)
	mock.ExpectSRem("logins:"+userid, "pad:1").SetErr(fmt.Errorf("some error"))
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusInternalServerError, sc, cookie)
	require.Equal(t, "/", loc)

	// something in login fails, doesn't matter what
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:" + userid).SetVal([]string{"pad:1"})
	mock.ExpectHDel("pad:1", userid, remote, redirect).SetVal(3)
	mock.ExpectSRem("logins:"+userid, "pad:1").SetVal(1)
	mock.ExpectSMembers("logins:userid").SetErr(fmt.Errorf("some error"))
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusInternalServerError, sc, cookie)
	require.Equal(t, "/", loc)

	// setting the pad in the login session fails
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetErr(fmt.Errorf("some error"))
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusInternalServerError, sc, cookie)
	require.Equal(t, "/", loc)

	// setting the pad in the login session returns wrong count
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(2)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusInternalServerError, sc, cookie)
	require.Equal(t, "/", loc)

	// redirect is missing
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid: userid,
		remote: remote,
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(1)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusFound, sc, cookie)
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)
	require.Equal(t, "/", loc)

	// redirect is empty
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		remote:   remote,
		redirect: "",
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(1)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusFound, sc, cookie)
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)
	require.Equal(t, "/", loc)

	// the happy path does a lot
	mock.ExpectHGetAll("pad:1").SetVal(map[string]string{
		userid:   userid,
		remote:   remote,
		redirect: redirect,
	})
	mock.ExpectSMembers("logins:userid").SetErr(redis.Nil)
	// the rest is from Test_Login happy path
	mock.ExpectSMembers("logins:userid").SetVal([]string{})
	mock.Regexp().ExpectHSet("token:.*", map[string]interface{}{
		userid: userid,
		remote: remote,
	}).SetVal(1)
	mock.Regexp().ExpectExpire("token:.*", expireme).SetVal(true)
	mock.Regexp().ExpectSAdd("logins:userid", "token:.*").SetVal(1)
	mock.Regexp().ExpectHSet("token:.*", "pad", "1").SetVal(1)
	loc, cookie, sc = v.LoginOTP(ctx, "1", remote)
	require.Equal(t, http.StatusFound, sc, cookie)
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)
	require.Equal(t, redirect, loc)
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

func Test_ValidateOTP(t *testing.T) {
	t.Parallel()

	db, mock := redismock.NewClientMock()
	l := logrus.WithField("test", "Test_ValidateOTP")
	v := NewValidator(db, cfg, l)
	exp := time.Duration(v.(*core).loginCookie("").MaxAge) * time.Second // hard way to get expiration

	// fails finding a login
	mock.ExpectHGetAll("token:token").SetErr(fmt.Errorf("some error"))
	uid, sc := v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, uid)

	// login uid missing
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{})
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusForbidden, sc)
	require.Empty(t, uid)

	// login uid empty
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: "",
	})
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusForbidden, sc)
	require.Empty(t, uid)

	// login pad missing
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
	})
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusForbidden, sc)
	require.Empty(t, uid)

	// login pad empty
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    "",
	})
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusForbidden, sc)
	require.Empty(t, uid)

	// remove pad fails
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    otp,
	})
	mock.ExpectHDel("token:token", otp).SetErr(fmt.Errorf("some error"))
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, uid)

	// remove pad update none
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    otp,
	})
	mock.ExpectHDel("token:token", otp).SetVal(0)
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, uid)

	// expire fails
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    otp,
	})
	mock.ExpectHDel("token:token", otp).SetVal(1)
	mock.ExpectExpire("token:token", exp).SetErr(fmt.Errorf("some error"))
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, uid)

	// expire count mismatch
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    otp,
	})
	mock.ExpectHDel("token:token", otp).SetVal(1)
	mock.ExpectExpire("token:token", exp).SetVal(false)
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Empty(t, uid)

	// happy path
	mock.ExpectHGetAll("token:token").SetVal(map[string]string{
		userid: userid,
		otp:    otp,
	})
	mock.ExpectHDel("token:token", otp).SetVal(1)
	mock.ExpectExpire("token:token", exp).SetVal(true)
	uid, sc = v.ValidateOTP(ctx, "token", otp)
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, shared.UUID(userid), uid)
}
