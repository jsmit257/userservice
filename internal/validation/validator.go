package valid

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	// use this for unit testing once the redis question is answered
	authz map[string]time.Time

	core struct {
		// this needs to be in redis, or something fast and shareable
		// would make a good unit mock
		authz
		// shoulen't need this with a proper remote store
		rwlock     sync.RWMutex
		timeout    time.Duration
		cookieName string
	}

	Validator interface {
		Clear(context.Context, shared.CID)
		Login(context.Context, shared.CID) *http.Cookie
		Logout(context.Context, string, shared.CID) (*http.Cookie, int)
		Valid(context.Context, string, shared.CID) int
	}
)

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

func NewValidator(cfg *config.Config) Validator {
	return &core{
		authz:      make(map[string]time.Time),
		rwlock:     sync.RWMutex{},
		timeout:    time.Duration(cfg.AuthzTimeout) * time.Minute,
		cookieName: cfg.CookieName,
	}
}

func (v *core) Clear(context.Context, shared.CID) {
	v.rwlock.Lock()
	defer v.rwlock.Unlock()

	for k := range v.authz {
		delete(v.authz, k)
	}
}

func (v *core) Login(ctx context.Context, cid shared.CID) *http.Cookie {
	result := &http.Cookie{
		Name:     v.cookieName,
		Value:    uuid.NewString(),
		Path:     "/",
		Expires:  time.Now().UTC().Add(v.timeout),
		HttpOnly: true,
	}

	v.rwlock.Lock()
	defer v.rwlock.Unlock()

	v.authz[result.Value] = time.Now().UTC()

	return result
}

func (v *core) Logout(ctx context.Context, token string, cid shared.CID) (*http.Cookie, int) {
	result := v.Valid(ctx, token, cid)
	if result == http.StatusFound {
		v.rwlock.Lock()
		defer v.rwlock.Unlock()

		delete(v.authz, token)
	}
	return logoutCookie(v.cookieName), result
}

func (v *core) Valid(ctx context.Context, token string, cid shared.CID) int {
	v.rwlock.Lock()
	defer v.rwlock.Unlock()

	if t, ok := v.authz[token]; !ok {
		return http.StatusForbidden
	} else if t.Add(v.timeout).Before(time.Now().UTC()) {
		return http.StatusForbidden
	}

	v.authz[token] = time.Now().UTC()

	return http.StatusFound
}
