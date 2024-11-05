package valid

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

func Test_Validator(t *testing.T) {

	ctx := context.Background()
	cid := shared.CID("Test_Validator")

	cfg := &config.Config{
		AuthzTimeout: 15,
		CookieName:   "foobar",
	}
	v := NewValidator(cfg)
	require.NotNil(t, v)

	valid := v.Login(ctx, cid)
	require.Equal(t, http.StatusFound, v.Valid(ctx, valid.Value, cid))
	require.Equal(t, http.StatusForbidden, v.Valid(ctx, "quux", cid))

	cookie, code := v.Logout(ctx, "quux", cid)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, logoutCookie(cfg.CookieName), cookie)

	cookie, code = v.Logout(ctx, valid.Value, cid)
	require.Equal(t, http.StatusFound, code)
	require.Equal(t, logoutCookie(cfg.CookieName), cookie)
	require.Equal(t, http.StatusForbidden, v.Valid(ctx, valid.Value, cid))

	valid = v.Login(ctx, cid)
	require.Equal(t, http.StatusFound, v.Valid(ctx, valid.Value, cid))
	v.Clear(ctx, cid)
	require.Equal(t, http.StatusForbidden, v.Valid(ctx, valid.Value, cid))

	temp := core{authz: map[string]time.Time{"foobar": time.Now().UTC().Add(-2 * time.Hour)}}
	require.Equal(t, http.StatusForbidden, temp.Valid(ctx, "foobar", cid))
}
