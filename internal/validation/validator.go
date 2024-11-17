package valid

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
)

const (
	userid = "userid"
	remote = "remote"
)

type (
	Validator interface {
		Login(context.Context, shared.UUID, string, shared.CID) (*http.Cookie, int)
		Logout(context.Context, string, shared.CID) (*http.Cookie, int)
		Valid(context.Context, string, shared.CID) (*http.Cookie, int)
	}

	authn interface {
		Exists(context.Context, ...string) *redis.IntCmd
		Expire(context.Context, string, time.Duration) *redis.BoolCmd
		HDel(context.Context, string, ...string) *redis.IntCmd
		HGet(context.Context, string, string) *redis.StringCmd
		HSet(context.Context, string, ...interface{}) *redis.IntCmd
		SAdd(context.Context, string, ...interface{}) *redis.IntCmd
		SMembers(context.Context, string) *redis.StringSliceCmd
		SRem(context.Context, string, ...interface{}) *redis.IntCmd
	}

	core struct {
		authn
		maxLogins    int
		logger       *logrus.Entry
		loginCookie  func() *http.Cookie
		logoutCookie *http.Cookie
	}
)

func NewValidator(client authn, cfg *config.Config, logger *logrus.Entry) Validator {
	genCookie := http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	}

	timeout := time.Duration(cfg.AuthnTimeout) * time.Minute

	return &core{
		authn:     client,
		maxLogins: cfg.MaxLogins,
		logger:    logger.WithField("pkg", "valid"),
		loginCookie: func() *http.Cookie {
			result := genCookie
			result.Value = uuid.NewString()
			result.Expires = time.Now().UTC().Add(timeout)
			result.MaxAge = int(cfg.AuthnTimeout * 60)
			return &result
		},
		logoutCookie: &genCookie,
	}
}

func (v *core) Login(ctx context.Context, uid shared.UUID, rmt string, cid shared.CID) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Login",
		"cid":      cid,
	})

	cookie := v.loginCookie()
	logins := "logins:" + string(uid)
	token := "token:" + cookie.Value

	if sc := v.checkCount(ctx, logins, cid); sc != http.StatusOK {
		l.WithField("status", sc).Error("check count fails")
		return nil, sc
	} else if err := v.authn.HSet(ctx, token, map[string]interface{}{
		userid: string(uid),
		remote: rmt,
	}).Err(); err != nil {
		l.WithError(err).Error("couldn't set new token")
		return nil, http.StatusInternalServerError
	} else if exp, err := v.authn.Expire(
		ctx,
		token,
		time.Duration(cookie.MaxAge)*time.Second,
	).Result(); err != nil {
		l.WithError(err).Error("failed to set expire on token")
		return nil, http.StatusInternalServerError
	} else if !exp {
		l.Error("no token was updated?")
		return nil, http.StatusInternalServerError
	} else if err = v.authn.SAdd(ctx, logins, "token:"+cookie.Value).Err(); err != nil {
		l.WithError(err).Error("couldn't add new login to current")
		return nil, http.StatusInternalServerError
	}

	return cookie, http.StatusOK
}

func (v *core) Logout(ctx context.Context, token string, cid shared.CID) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Logout",
		"cid":      cid,
	})

	key := "token:" + token

	if _, result := v.Valid(ctx, token, cid); result != http.StatusFound {
		l.WithField("status code", "result").Error("logout request isn't valid")
		return nil, result
	} else if uid, err := v.authn.HGet(ctx, key, userid).Result(); err != nil && err != redis.Nil {
		l.WithError(err).Error("couldn't get userid for token")
		return nil, http.StatusInternalServerError
	} else if err == redis.Nil {
		l.WithError(err).Error("user isn't logged in")
		return nil, http.StatusForbidden
	} else if err := v.authn.HDel(ctx, key, userid, remote).Err(); err != nil {
		l.WithError(err).Error("couldn't remove token")
		return nil, http.StatusInternalServerError
	} else if err = v.authn.SRem(ctx, "logins:"+uid, key).Err(); err != nil {
		l.WithError(err).Error("couldn't remove auth token from user")
		return nil, http.StatusInternalServerError
	}

	return v.logoutCookie, http.StatusAccepted // liked StatusGone better, but it's a 4xx series
}

func (v *core) Valid(ctx context.Context, token string, cid shared.CID) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Valid",
		"cid":      cid,
	})

	key := "token:" + token
	cookie := v.loginCookie()
	cookie.Value = token

	if count, err := v.authn.Exists(ctx, key).Result(); err != nil {
		l.WithError(err).Error("checking exists")
		return nil, http.StatusInternalServerError
	} else if count == 0 {
		l.Error("token doesn't exist")
		return nil, http.StatusForbidden
	} else if found, err := v.authn.Expire(ctx, key, time.Duration(cookie.MaxAge)*time.Second).Result(); err != nil {
		l.WithError(err).Error("setting new expiry")
		return nil, http.StatusInternalServerError
	} else if !found {
		l.Error("no token was updated?")
		return nil, http.StatusInternalServerError
	}

	return cookie, http.StatusFound
}

func (v *core) checkCount(ctx context.Context, key string, cid shared.CID) int {
	l := v.logger.WithFields(logrus.Fields{
		"function": "checkCount",
		"cid":      cid,
	})

	logins, err := v.authn.SMembers(ctx, key).Result()
	if err != nil {
		l.WithError(err).Error("failed getting smembers")
		return http.StatusInternalServerError
	} else if len(logins) < v.maxLogins {
		return http.StatusOK
	}

	var expired []interface{}
	for _, token := range logins {
		exists, err := v.authn.Exists(ctx, token).Result()
		if err != nil {
			l.WithError(err).Error("failed checking for token")
			return http.StatusInternalServerError
		} else if exists == 0 {
			l.Debug("found expired token")
			expired = append(expired, token)
		}
	}

	if len(expired) == 0 {
		return http.StatusTooManyRequests
	} else if removed, err := v.authn.SRem(ctx, key, expired...).Result(); err != nil {
		l.
			WithError(err).
			WithField("expired", expired).
			Error("couldn't remove stale tokens")
		return http.StatusInternalServerError
	} else if len(logins)-int(removed) < v.maxLogins {
		return http.StatusOK
	}

	return http.StatusTooManyRequests
}
