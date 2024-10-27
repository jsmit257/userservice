package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

func Test_ContactUpdate(t *testing.T) {
	tcs := map[string]struct {
		contactid shared.UUID
		send      shared.Contact
		resp      string
		sc        int
	}{
		"happy_path": {
			contactid: users[contactpatch].UUID,
			send: func(c shared.Contact) shared.Contact {
				c.FirstName = "new firstname"
				c.BillTo = addresses["address_3"]

				return c
			}(*users[contactpatch].Contact),
			sc: http.StatusOK,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%d/contact/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.contactid),
				contactToReader(&tc.send))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, string(body))
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func contactToReader(c *shared.Contact) io.Reader {
	body, _ := json.Marshal(c)
	return bytes.NewReader(body)
}
