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
		Login(context.Context, shared.UUID, string) (*http.Cookie, int)
		Logout(context.Context, string) (*http.Cookie, int)
		Valid(context.Context, string) (*http.Cookie, int)
		OTP(context.Context, shared.UUID, string) (string, int)
		ValidOTP(context.Context, string, string) (*http.Cookie, int)
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

	return &core{
		authn:     client,
		maxLogins: cfg.MaxLogins,
		logger:    logger.WithField("pkg", "valid"),
		loginCookie: func() *http.Cookie {
			result := genCookie
			result.Value = uuid.NewString()
			result.Expires = time.Now().UTC().Add(time.Duration(cfg.AuthnTimeout) * time.Minute)
			result.MaxAge = int(cfg.AuthnTimeout * 60)
			return &result
		},
		logoutCookie: &genCookie,
	}
}

func (v *core) Login(ctx context.Context, uid shared.UUID, rmt string) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Login",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	cookie := v.loginCookie()
	logins := "logins:" + string(uid)
	token := "token:" + cookie.Value

	if sc := v.checkCount(ctx, logins); sc != http.StatusOK {
		l.WithField("status", sc).Error("check count fails")
		return v.logoutCookie, sc
	} else if err := v.authn.HSet(ctx, token, map[string]interface{}{
		userid: string(uid),
		remote: rmt,
	}).Err(); err != nil {
		l.WithError(err).Error("couldn't set new token")
		return v.logoutCookie, http.StatusInternalServerError
	} else if exp, err := v.authn.Expire(
		ctx,
		token,
		time.Duration(cookie.MaxAge)*time.Second,
	).Result(); err != nil {
		l.WithError(err).Error("failed to set expire on token")
		return v.logoutCookie, http.StatusInternalServerError
	} else if !exp {
		l.Error("no token was updated?")
		return v.logoutCookie, http.StatusInternalServerError
	} else if err = v.authn.SAdd(ctx, logins, token).Err(); err != nil {
		l.WithError(err).Error("couldn't add new login to current")
		return v.logoutCookie, http.StatusInternalServerError
	}

	return cookie, http.StatusOK
}

func (v *core) Logout(ctx context.Context, token string) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Logout",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	key := "token:" + token

	if _, result := v.Valid(ctx, token); result != http.StatusFound {
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

	return v.logoutCookie, http.StatusNoContent // liked StatusGone better, but it's a 4xx series
}

func (v *core) Valid(ctx context.Context, token string) (*http.Cookie, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "Valid",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	key := "token:" + token
	cookie := v.loginCookie()
	cookie.Value = token

	if count, err := v.authn.Exists(ctx, key).Result(); err != nil {
		l.WithError(err).Error("checking exists")
		return v.logoutCookie, http.StatusInternalServerError
	} else if count == 0 {
		l.Error("token doesn't exist")
		return v.logoutCookie, http.StatusForbidden
	} else if found, err := v.authn.Expire(ctx, key, time.Duration(cookie.MaxAge)*time.Second).Result(); err != nil {
		l.WithError(err).Error("setting new expiry")
		return v.logoutCookie, http.StatusInternalServerError
	} else if !found {
		l.Error("no token was updated?")
		return v.logoutCookie, http.StatusInternalServerError
	}

	return cookie, http.StatusFound
}

func (v *core) OTP(ctx context.Context, uid shared.UUID, rmt string) (string, int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "OTP",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	pad := uuid.NewString()
	key := "pad:" + pad
	if err := v.authn.HSet(ctx, key, map[string]interface{}{
		userid: uid,
		remote: rmt,
	}).Err(); err != nil {
		l.WithError(err).Error("creating pad entry")
		return err.Error(), http.StatusInternalServerError
	} else if err := v.authn.Expire(ctx, key, 15*time.Minute).Err(); err != nil {
		l.WithError(err).Error("expiring new pad entry")
		return err.Error(), http.StatusInternalServerError
	} else if n, err := v.authn.SAdd(ctx, "logins:"+string(uid), key).Result(); err != nil {
		l.WithError(err).Error("expiring new pad entry")
		return err.Error(), http.StatusInternalServerError
	} else if n != 1 {
		l.Error("no row was updated")
		return "no row was updated", http.StatusInternalServerError
	}

	return pad, http.StatusOK
}

func (v *core) ValidOTP(ctx context.Context, pad string, rmt string) (cookie *http.Cookie, sc int) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "CheckOTP",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	key := "pad:" + pad
	if result, err := v.authn.HGetAll(ctx, key).Result(); err != nil {
		l.WithError(err).Error(err.Error())
		return v.logoutCookie, http.StatusForbidden
	} else if uid, ok := result[userid]; !ok {
		l.Error("missing uid")
		return v.logoutCookie, http.StatusForbidden
	} else if uid == "" {
		l.Error("empty uid")
		return v.logoutCookie, http.StatusForbidden
	} else if sc := v.clearLogins(ctx, shared.UUID(result[userid])); sc != http.StatusGone {
		return v.logoutCookie, sc
	} else if cookie, sc = v.Login(ctx, shared.UUID(result[userid]), rmt); sc != http.StatusOK {
		return v.logoutCookie, sc
	}

	return cookie, http.StatusFound
}

func (v *core) clearTokens(ctx context.Context, tokens []string) ([]interface{}, error) {
	l := v.logger.WithFields(logrus.Fields{
		"function": "clearTokens",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	var result []interface{}
	for _, t := range tokens {
		if err := v.authn.HDel(ctx, t, userid, remote).Err(); err != nil {
			l.WithError(err).Error("couldn't clear all tokens") // what if it just expired?
			return result, err
		} else {
			result = append(result, t)
		}
	}

	return result, nil
}

func (v *core) clearLogins(ctx context.Context, uid shared.UUID) int {
	l := v.logger.WithFields(logrus.Fields{
		"function": "clearLogins",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	key := "logins:" + string(uid)
	if tokens, err := v.authn.SMembers(ctx, key).Result(); err != nil && err != redis.Nil {
		l.WithError(err).Error("couldn't get userid for token")
		return http.StatusForbidden
	} else if err == redis.Nil {
		l.Error("user isn't logged in")
		return http.StatusGone
	} else if len(tokens) == 0 { // redundant?
		l.Error("no tokens to clear")
		return http.StatusGone
	} else if els, err := v.clearTokens(ctx, tokens); err != nil {
		return http.StatusInternalServerError
	} else if err = v.authn.SRem(ctx, key, els...).Err(); err != nil {
		l.WithError(err).Error("couldn't remove logins")
		return http.StatusInternalServerError
	}

	return http.StatusGone
}

func (v *core) checkCount(ctx context.Context, key string) int {
	l := v.logger.WithFields(logrus.Fields{
		"function": "checkCount",
		"cid":      ctx.Value(shared.CTXKey("cid")).(shared.CID),
	})

	tokens, err := v.authn.SMembers(ctx, key).Result()
	if err != nil {
		l.WithError(err).Error("failed getting smembers")
		return http.StatusInternalServerError
	} else if len(tokens) < v.maxLogins {
		return http.StatusOK
	}

	var expired []interface{}
	for _, t := range tokens {
		exists, err := v.authn.Exists(ctx, t).Result()
		if err != nil {
			l.WithError(err).Error("failed checking for token")
			return http.StatusInternalServerError
		} else if exists == 0 {
			l.Debug("found expired token")
			expired = append(expired, t)
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
	} else if len(tokens)-int(removed) < v.maxLogins {
		return http.StatusOK
	}

	return http.StatusTooManyRequests
}
