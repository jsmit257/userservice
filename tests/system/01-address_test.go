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

var addresses = make(map[string]*shared.Address, 5)

func Test_AddressInit(t *testing.T) {
	for name, addr := range addresses {
		name, addr := name, addr

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/address/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					addr.UUID),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			require.Nil(t, json.Unmarshal(body, addresses[name]))
		})
	}
}

func Test_AddressesGet(t *testing.T) {
	// t.Parallel() // this can't run before patch

	tcs := map[string]struct {
		resp string
		sc   int
	}{"happy_path": {sc: http.StatusOK}}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/addresses",
					cfg.ServerHost,
					cfg.ServerPort),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode)
			if resp.StatusCode != http.StatusOK {
				require.Equal(t, tc.resp, string(body))
				return
			}

			var list []shared.Address
			require.Nil(t, json.Unmarshal(body, &list))
			alladdresses := func() []shared.Address {
				var result []shared.Address
				for _, a := range addresses {
					result = append(result, *a)
				}
				return result
			}()
			require.Subset(t, alladdresses, list)
		})
	}
}

func Test_AddressGet(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		addressid shared.UUID
		resp      *shared.Address
		sc        int
	}{
		// "happy_path": {}, // redundant
		"get_fails": {
			addressid: "users.readonly.UUID",
			sc:        http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/address/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.addressid),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode)

			var got *shared.Address
			if resp.StatusCode == http.StatusOK {
				require.Nil(t, json.Unmarshal(body, &got))
			}
			require.Equal(t, tc.resp, got)
		})
	}
}

func Test_AddressPost(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		send shared.Address
		resp string
		sc   int
	}{
		// "happy_path": {}, // covered in seed_test.go
		// "error": {}, // what error?
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%d/address", cfg.ServerHost, cfg.ServerPort),
				addressToReader(&tc.send))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, "body: '%s'", body)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_AddressPatch(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		addressid shared.UUID
		send      shared.Address
		resp      string
		sc        int
	}{
		"happy_path": {
			addressid: addresses["address_patch"].UUID,
			send: shared.Address{
				UUID:    addresses["address_patch"].UUID,
				Street2: "roof",
			},
			sc: http.StatusNoContent,
		},
		"not_found": {
			addressid: "not-found",
			send:      shared.Address{UUID: "somebody else"},
			resp:      shared.AddressNotUpdatedError.Error(),
			sc:        http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%d/address/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.addressid),
				addressToReader(&tc.send))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func addressToReader(a *shared.Address) io.Reader {
	body, _ := json.Marshal(a)
	return bytes.NewReader(body)
}
