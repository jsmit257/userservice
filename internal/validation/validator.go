package valid

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/shared/v1"
)

const (
	userid   = "userid"
	remote   = "remote"
	otp      = "pad"
	redirect = "redirect"
)

type (
	Validator interface {
		Login(context.Context, shared.UUID, string) (*http.Cookie, int)
		Logout(context.Context, string) (*http.Cookie, int)
		Valid(context.Context, string) (*http.Cookie, int)
		OTP(context.Context, shared.UUID, string, string) (string, int)
		LoginOTP(context.Context, string, string) (string, *http.Cookie, int)
		ValidateOTP(context.Context, string, string) (shared.UUID, int)
	}

	authn interface {
		Exists(context.Context, ...string) *redis.IntCmd
		Expire(context.Context, string, time.Duration) *redis.BoolCmd
		HDel(context.Context, string, ...string) *redis.IntCmd
		HGet(context.Context, string, string) *redis.StringCmd
		HGetAll(context.Context, string) *redis.MapStringStringCmd
		HSet(context.Context, string, ...interface{}) *redis.IntCmd
		SAdd(context.Context, string, ...interface{}) *redis.IntCmd
		SMembers(context.Context, string) *redis.StringSliceCmd
		SRem(context.Context, string, ...interface{}) *redis.IntCmd
	}

	core struct {
		authn
		maxLogins    int
		log          *logrus.Entry
		metrics      *prometheus.CounterVec
		loginCookie  func(string) *http.Cookie
		logoutCookie *http.Cookie
	}
)

var NotAuthorized = fmt.Errorf("not authorized")

func NewValidator(client authn, cfg *config.Config, logger *logrus.Entry) Validator {
	genCookie := http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	}

	return &core{
		authn:     client,
		maxLogins: cfg.MaxLogins,
		log: logger.WithFields(logrus.Fields{
			"pkg": "valid",
			"db":  "redis",
		}),
		metrics: metrics.DataMetrics.MustCurryWith(prometheus.Labels{
			"db":  "redis",
			"pkg": "valid",
		}),
		loginCookie: func(v string) *http.Cookie {
			result := genCookie
			result.Value = v
			result.Expires = time.Now().UTC().Add(time.Duration(cfg.AuthnTimeout) * time.Minute)
			result.MaxAge = int(cfg.AuthnTimeout * 60)
			return &result
		},
		logoutCookie: &genCookie,
	}
}

func (v *core) Login(ctx context.Context, uid shared.UUID, rmt string) (*http.Cookie, int) {
	t := v.tracker(ctx, "Login")

	cookie := v.loginCookie(uuid.NewString())
	logins := "logins:" + string(uid)
	token := "token:" + cookie.Value

	if code := v.checkCount(ctx, logins); code != http.StatusOK {
		return v.logoutCookie, t.sc(code).
			err(fmt.Errorf("too many logins")).
			done("check count fails").
			sc()
	} else if err := v.authn.HSet(ctx, token, map[string]interface{}{
		userid: string(uid),
		remote: rmt,
	}).Err(); err != nil {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("couldn't set new token").
			sc()
	} else if exp, err := v.authn.Expire(
		ctx,
		token,
		time.Duration(cookie.MaxAge)*time.Second,
	).Result(); err != nil {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("failed to set expire on token").
			sc()
	} else if !exp {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("no token was updated?")).
			done("no token was updated?").
			sc()
	} else if err = v.authn.SAdd(ctx, logins, token).Err(); err != nil {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("couldn't add new login to current").
			sc()
	}

	return cookie, t.sc(http.StatusOK).ok().sc()
}

func (v *core) Logout(ctx context.Context, token string) (*http.Cookie, int) {
	t := v.tracker(ctx, "Logout")

	key := "token:" + token
	if _, code := v.Valid(ctx, token); code != http.StatusFound {
		return nil, t.sc(code).err(NotAuthorized).done("logout request isn't valid").sc()
	} else if uid, err := v.authn.HGet(ctx, key, userid).Result(); err != nil && err != redis.Nil {
		return nil, t.sc(http.StatusInternalServerError).
			err(err).
			done("couldn't get userid for token").
			sc()
	} else if err == redis.Nil {
		return nil, t.sc(http.StatusForbidden).
			err(NotAuthorized).
			done("user isn't logged in").
			sc()
	} else if err := v.authn.HDel(ctx, key, userid, remote).Err(); err != nil {
		return nil, t.sc(http.StatusInternalServerError).
			err(err).
			done("couldn't remove token").
			sc()
	} else if err = v.authn.SRem(ctx, "logins:"+uid, key).Err(); err != nil {
		return nil, t.sc(http.StatusInternalServerError).
			err(err).
			done("couldn't remove auth token from user").
			sc()
	}

	return v.logoutCookie, t.sc(http.StatusNoContent).ok().sc() // liked StatusGone better, but it's a 4xx series
}

func (v *core) Valid(ctx context.Context, token string) (*http.Cookie, int) {
	t := v.tracker(ctx, "Valid")

	cookie := v.loginCookie(token)
	key := "token:" + token
	if count, err := v.authn.Exists(ctx, key).Result(); err != nil {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("checking exists").
			sc()
	} else if count == 0 {
		return v.logoutCookie, t.sc(http.StatusForbidden).
			err(fmt.Errorf("token doesn't exist")).
			done("token doesn't exist").
			sc()
	} else if found, err := v.authn.Expire(ctx, key, time.Duration(cookie.MaxAge)*time.Second).Result(); err != nil {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("setting new expiry").
			sc()
	} else if !found {
		return v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("token doesn't exist")).
			done("no token was updated?").
			sc()
	}

	return cookie, t.sc(http.StatusFound).ok().sc()
}

func (v *core) OTP(ctx context.Context, uid shared.UUID, rmt, three02 string) (string, int) {
	t := v.tracker(ctx, "OTP")

	pad := uuid.NewString()
	key := "pad:" + pad
	if err := v.authn.HSet(ctx, key, map[string]interface{}{
		userid:   uid,
		remote:   rmt,
		redirect: three02,
	}).Err(); err != nil {
		return err.Error(), t.sc(http.StatusInternalServerError).
			err(err).
			done("creating pad entry").
			sc()
	} else if err := v.authn.Expire(ctx, key, 15*time.Minute).Err(); err != nil {
		return err.Error(), t.sc(http.StatusInternalServerError).
			err(err).
			done("expiring new pad entry").
			sc()
	} else if n, err := v.authn.SAdd(ctx, "logins:"+string(uid), key).Result(); err != nil {
		return err.Error(), t.sc(http.StatusInternalServerError).
			err(err).
			done("expiring new pad entry").
			sc()
	} else if n != 1 {
		return "no row was updated", t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("wrong number of rows updated: %d", n)).
			done("no row was updated").
			sc()
	} else {
		t.fields(logrus.Fields{"pad": pad}).warn("remove this")
	}

	return pad, t.sc(http.StatusOK).ok().sc()
}

func (v *core) LoginOTP(ctx context.Context, pad, rmt string) (location string, cookie *http.Cookie, code int) {
	t := v.tracker(ctx, "LoginOTP")
	three02 := "/"

	key := "pad:" + pad
	if result, err := v.authn.HGetAll(ctx, key).Result(); err != nil {
		return three02, v.logoutCookie, t.sc(http.StatusForbidden).err(err).done(err.Error()).sc()
	} else if uid, ok := result[userid]; !ok {
		return three02, v.logoutCookie, t.sc(http.StatusForbidden).err(NotAuthorized).done("missing uid").sc()
	} else if uid == "" {
		return three02, v.logoutCookie, t.sc(http.StatusForbidden).err(NotAuthorized).done("empty uid").sc()
	} else if code := v.clearLogins(ctx, shared.UUID(result[userid])); code != http.StatusGone {
		return three02, v.logoutCookie, t.sc(code).done("clearing logins").sc()
	} else if cookie, code = v.Login(ctx, shared.UUID(result[userid]), rmt); code != http.StatusOK {
		return three02, v.logoutCookie, t.sc(code).done("logging in").sc()
	} else if count, err := v.authn.HSet(ctx, "token:"+cookie.Value, "pad", pad).Result(); err != nil {
		return three02, v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(err).
			done("setting pad attribute").
			sc()
	} else if count != 1 {
		return three02, v.logoutCookie, t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("wrong number of fields updated: %d", count)).
			done("setting pad attribute").
			sc()
	} else if temp, ok := result[redirect]; !ok {
		t.warn("redirect is missing")
	} else if temp == "" {
		t.warn("redirect is empty") // for now this is somebody else's problem
	} else {
		three02 = temp
	}

	return three02, cookie, t.sc(http.StatusFound).ok().sc()
}

func (v *core) ValidateOTP(ctx context.Context, token, pad string) (shared.UUID, int) {
	var result shared.UUID

	t := v.tracker(ctx, "ValidateOTP")

	exp := time.Duration(v.loginCookie("").MaxAge) * time.Second // hard way to get expiration
	key := "token:" + token
	if login, err := v.authn.HGetAll(ctx, key).Result(); err != nil {
		return result, t.sc(http.StatusInternalServerError).
			err(err).
			done("getting login").
			sc()
	} else if uid, ok := login[userid]; !ok {
		return result, t.sc(http.StatusForbidden).
			err(fmt.Errorf("token is missing userid")).
			done("invalid token").
			sc()
	} else if uid == "" {
		return result, t.sc(http.StatusForbidden).
			err(fmt.Errorf("token userid is empty")).
			done("invalid token").
			sc()
	} else if p, ok := login[otp]; !ok {
		return result, t.sc(http.StatusForbidden).
			err(fmt.Errorf("not a OTP login")).
			done("invalid request").
			sc()
	} else if p != pad {
		return result, t.sc(http.StatusForbidden).
			err(fmt.Errorf("pads don't match")).
			done("invalid request").
			sc()
	} else if count, err := v.authn.HDel(ctx, key, otp).Result(); err != nil {
		return result, t.sc(http.StatusInternalServerError).
			err(err).
			done("updating login").
			sc()
	} else if count != 1 {
		return result, t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("wrong number of attributes removed: %d", count)).
			done("updating login").
			sc()
	} else if found, err := v.authn.Expire(ctx, key, exp).Result(); err != nil {
		return result, t.sc(http.StatusInternalServerError).
			err(err).
			done("setting new expiry").
			sc()
	} else if !found {
		return result, t.sc(http.StatusInternalServerError).
			err(fmt.Errorf("token doesn't exist")).
			done("no token was updated?").
			sc()
	} else {
		result = shared.UUID(uid)
	}

	return result, http.StatusOK
}

func (v *core) clearTokens(ctx context.Context, tokens []string) ([]interface{}, error) {
	t := v.tracker(ctx, "clearLogins")

	var result []interface{}
	for _, token := range tokens {
		if err := v.authn.HDel(ctx, token, userid, remote).Err(); err != nil {
			return result, t.err(err).done("couldn't clear all tokens").err() // what if it just expired?
		} else {
			result = append(result, token)
		}
	}

	return result, t.ok().err()
}

func (v *core) clearLogins(ctx context.Context, uid shared.UUID) int {
	t := v.tracker(ctx, "clearLogins")

	key := "logins:" + string(uid)
	if tokens, err := v.authn.SMembers(ctx, key).Result(); err != nil && err != redis.Nil {
		return t.sc(http.StatusForbidden).err(err).done("couldn't get userid for token").sc()
	} else if err == redis.Nil {
		return t.sc(http.StatusGone).err(NotAuthorized).done("user isn't logged in").sc()
	} else if l := len(tokens); l == 0 { // redundant?
		return t.sc(http.StatusGone).err(NotAuthorized).done("no tokens to clear").sc()
	} else if els, err := v.clearTokens(ctx, tokens); err != nil {
		return t.sc(http.StatusInternalServerError).err(err).done("clearing tokens").sc()
	} else if err = v.authn.SRem(ctx, key, els...).Err(); err != nil {
		return t.sc(http.StatusInternalServerError).err(err).done("couldn't remove logins").sc()
	} else {
		t.fields(logrus.Fields{"count": l})
	}

	return t.sc(http.StatusGone).ok().sc()
}

func (v *core) checkCount(ctx context.Context, key string) int {
	t := v.tracker(ctx, "checkCount")

	tokens, err := v.authn.SMembers(ctx, key).Result()
	if err != nil {
		return t.sc(http.StatusInternalServerError).
			err(err).
			done("failed getting smembers").
			sc()
	} else if len(tokens) < v.maxLogins {
		return t.sc(http.StatusOK).ok().sc()
	}

	var expired []interface{}
	for _, token := range tokens {
		exists, err := v.authn.Exists(ctx, token).Result()
		if err != nil {
			return t.sc(http.StatusInternalServerError).
				err(err).
				done("failed checking for token").
				sc()
		} else if exists == 0 {
			t.debug("found expired token")
			expired = append(expired, token)
		}
	}

	if len(expired) == 0 {
		return t.sc(http.StatusTooManyRequests).done("too many logins").sc()
	} else if removed, err := v.authn.SRem(ctx, key, expired...).Result(); err != nil {
		return t.sc(http.StatusInternalServerError).
			err(err).
			fields(logrus.Fields{"expired": expired}).
			done("couldn't remove stale tokens").
			sc()
	} else if len(tokens)-int(removed) < v.maxLogins {
		return t.sc(http.StatusOK).ok().sc()
	}

	return t.sc(http.StatusTooManyRequests).done("too many logins").sc()
}

func (v *core) tracker(ctx context.Context, fn string) tracker {
	cid := ctx.Value(shared.CTXKey("cid")).(shared.CID)

	l := v.log.WithFields(logrus.Fields{
		"function": fn,
		"cid":      cid,
	})
	l.Info("starting work")

	return &track{
		l: l,
		m: v.metrics.MustCurryWith(prometheus.Labels{"function": fn}),
		r: &trackresults{},
		s: time.Now().UTC(),
	}
}

type trackresults struct {
	code int
	e    error
}

type resulter interface {
	err() error
	sc() int
}

func (r *trackresults) err() error {
	return r.e
}
func (r *trackresults) sc() int {
	return r.code
}

type track struct {
	l *logrus.Entry
	m *prometheus.CounterVec
	r *trackresults
	s time.Time
}

type tracker interface {
	debug(string, ...any) tracker
	warn(string, ...any) tracker
	done(string) resulter
	err(error) tracker
	fields(f logrus.Fields) tracker
	ok() resulter
	sc(code int) tracker
}

func (t *track) fields(f logrus.Fields) tracker {
	return &track{
		l: t.l.WithFields(f),
		m: t.m,
		r: t.r,
		s: t.s,
	}
}
func (t *track) sc(code int) tracker {
	return &track{
		l: t.l.WithField("statuscode", code),
		m: t.m,
		r: &trackresults{code: code, e: t.r.e},
		s: t.s,
	}
}
func (t *track) err(e error) tracker {
	if t.r.e != nil {
		// if we don't do this MustCurryWith (probably?) will; don't really want to return an error
		panic(fmt.Errorf("error is already set for tracker (old: %w), (new: %w)", t.r.e, e))
	}

	return &track{
		l: t.l.WithError(e),
		m: t.m.MustCurryWith(prometheus.Labels{"status": e.Error()}),
		r: &trackresults{code: t.r.code, e: e},
		s: t.s,
	}
}
func (t *track) done(msg string) resulter {
	var status []string
	var closer = t.l.WithField("duration", time.Since(t.s).String()).Info
	if t.r.e == nil {
		status = []string{"ok"}
	} else {
		closer = t.l.Error
	}
	t.m.WithLabelValues(status...).Inc()
	closer(msg)
	return t.r
}
func (t *track) ok() resulter {
	return t.done("finished work")
}
func (t *track) debug(msg string, args ...any) tracker {
	t.l.Debugf(msg, args...)
	return t
}
func (t *track) warn(msg string, args ...any) tracker {
	t.l.Warnf(msg, args...)
	return t
}
