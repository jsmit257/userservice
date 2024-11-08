package valid

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	authn struct {
		UserID  shared.UUID `json:"userid,omitempty"`
		Expires time.Time   `json:"expires,omitempty"`
		Remote  string      `json:"remote,omitempty"`
	}

	authnClient interface {
		Decr(context.Context, string) *redis.IntCmd
		HDel(context.Context, string, ...string) *redis.IntCmd
		HGet(context.Context, string, string) *redis.StringCmd
		HSet(context.Context, string, ...interface{}) *redis.IntCmd
		Incr(context.Context, string) *redis.IntCmd
	}

	core struct {
		rc         authnClient
		timeout    time.Duration
		cookieName string
	}

	Validator interface {
		Login(context.Context, shared.UUID, string, shared.CID) (*http.Cookie, int)
		Logout(context.Context, string, shared.CID) (*http.Cookie, int)
		Valid(context.Context, string, shared.CID) int
	}
)

func (a authn) val() map[string]interface{} {
	text, _ := json.Marshal(a)
	var result map[string]interface{}
	_ = json.Unmarshal(text, &result)
	return result
}

func logoutCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	}
}

func NewValidator(client authnClient, cfg *config.Config) Validator {
	return &core{
		rc:         client,
		timeout:    time.Duration(cfg.AuthnTimeout) * time.Minute,
		cookieName: cfg.CookieName,
	}
}

func (v *core) Login(ctx context.Context, userid shared.UUID, remote string, cid shared.CID) (*http.Cookie, int) {
	key := "user:" + string(userid)
	if count, err := v.rc.Incr(ctx, key).Result(); err != nil && err != redis.Nil {
		return nil, http.StatusInternalServerError
	} else if count > 5 { // FIXME: move magic to config
		if err = v.rc.Decr(ctx, key).Err(); err != nil {
			return nil, http.StatusInternalServerError
		}
		return nil, http.StatusTooManyRequests
	}

	result := &http.Cookie{
		Name:     v.cookieName,
		Value:    uuid.NewString(),
		Path:     "/",
		Expires:  time.Now().UTC().Add(v.timeout),
		HttpOnly: true,
	}

	err := v.rc.HSet(ctx, "token:"+result.Value, authn{
		UserID:  userid,
		Expires: result.Expires,
		Remote:  remote,
	}.val()).Err()
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	return result, http.StatusOK
}

func (v *core) Logout(ctx context.Context, token string, cid shared.CID) (*http.Cookie, int) {
	if result := v.Valid(ctx, token, cid); result != http.StatusFound {
		return nil, result
	}

	if userid, err := v.rc.HGet(ctx, "token:"+token, "userid").Result(); err != nil {
		return nil, http.StatusInternalServerError
	} else if err := v.rc.HDel(ctx, "token:"+token, "userid", "expires", "remote").Err(); err != nil {
		return nil, http.StatusInternalServerError
	} else if err = v.rc.Decr(ctx, "user:"+userid).Err(); err != nil {
		return nil, http.StatusInternalServerError
	}

	return logoutCookie(v.cookieName), http.StatusAccepted // like StatusGone better, but it's a 4xx series
}

func (v *core) Valid(ctx context.Context, token string, cid shared.CID) int {
	key := "token:" + token
	if expires, err := v.rc.HGet(ctx, key, "expires").Result(); err == redis.Nil {
		return http.StatusForbidden
	} else if err != nil {
		return http.StatusInternalServerError
	} else if t, err := time.Parse(time.RFC3339, expires); err != nil {
		return http.StatusInternalServerError
	} else if t.Add(v.timeout).Before(time.Now().UTC()) {
		return http.StatusForbidden
	} else if _, err := v.rc.HSet(ctx, key, "expires", time.Now().UTC().Add(v.timeout)).Result(); err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusFound
}
