package seed

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/shared/v1"
)

const (
	usernocontact = iota
	useremptycontact
	userbillto
	usershipto
	userbilltoshipto
	readonly
	userpatch
	userdelete
	contactpatch
	resetpass
	logintest
	logouttest
	pwdtest
	passchange
	pwdfail
	logindelete
	last // leave this at the end and probably don't use it
)

var (
	cfg   = config.NewConfig()
	users = make([]*shared.User, last)
)

func Test_SeedUsers(t *testing.T) {
	for i, j := 0, last; i < j; i++ {
		i := i
		name := fmt.Sprintf("user_%d", i)
		email := shared.Email(name)
		cell := shared.Cell(name)

		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%d/user", cfg.ServerHost, cfg.ServerPort),
				userToReader(&shared.User{
					Name:  name,
					Email: &email,
					Cell:  &cell,
				}))
			require.Nil(t, err)

			resp, err := (&http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}).Do(req)
			require.Nil(t, err)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, http.StatusCreated, resp.StatusCode, "body: '%s'", body)
			require.NotEmpty(t, body)

			users[i] = &shared.User{UUID: shared.UUID(body)}
		})
	}
}

func Test_SeedAddresses(t *testing.T) {
	tcs := map[string]struct {
		postdata shared.Address
		sc       int
	}{
		"address_1": {
			postdata: shared.Address{
				Street1: "street 1/1",
				Street2: "street 2/1",
				City:    "laketown",
				State:   "Mississippi",
				Country: "US",
				Zip:     "00123",
			},
			sc: http.StatusOK,
		},
		"address_2": {
			postdata: shared.Address{
				Street1: "street 1/2",
				Street2: "",
				City:    "beantown",
				State:   "MA",
				Country: "United States",
				Zip:     "98765",
			},
			sc: http.StatusOK,
		},
		"address_3": {
			postdata: shared.Address{
				Street1: "street 1/3",
				Street2: "street 2/3",
				City:    "little korea",
				State:   "CA",
				Country: "Here",
				Zip:     "666",
			},
			sc: http.StatusOK,
		},
		"walla_walla": {
			postdata: shared.Address{
				Street1: "111 walla walla st",
				Street2: "",
				City:    "walla walla",
				State:   "washington",
				Country: "united states",
				Zip:     "11111",
			},
			sc: http.StatusOK,
		},
		"address_patch": {
			postdata: shared.Address{
				Street1: "lakeshore dr",
				Street2: "",
				City:    "arlington",
				State:   "VA",
				Country: "united states",
				Zip:     "10101",
			},
			sc: http.StatusOK,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%s/address", "localhost", "3000"),
				addressToReader(&tc.postdata))
			require.Nil(t, err)

			resp, err := (&http.Client{}).Do(req)
			require.Nil(t, err)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, "body: '%s'", body)

			addresses[name] = &shared.Address{UUID: shared.UUID(body)}
		})
	}
}

func Test_SeedContacts(t *testing.T) {
	tcs := map[string]struct {
		postdata shared.Contact
		user     *shared.User
		sc       int
	}{
		"no_shipto_no_billto": {
			postdata: shared.Contact{
				FirstName: "john",
				LastName:  "smith",
				BillTo:    nil,
				ShipTo:    nil,
			},
			user: users[useremptycontact],
			sc:   http.StatusOK,
		},
		"shipto_no_billto": {
			postdata: shared.Contact{
				FirstName: "jenny",
				LastName:  "tupelo",
				BillTo:    nil,
				ShipTo:    addresses["address_1"],
			},
			user: users[usershipto],
			sc:   http.StatusOK,
		},
		"billto_no_shipto": {
			postdata: shared.Contact{
				FirstName: "jerkface",
				LastName:  "cracker",
				BillTo:    addresses["address_2"],
				ShipTo:    nil,
			},
			user: users[userbillto],
			sc:   http.StatusOK,
		},
		"billto_shipto": {
			postdata: shared.Contact{
				FirstName: "ko",
				LastName:  "walla",
				BillTo:    addresses["walla_walla"],
				ShipTo:    addresses["walla_walla"],
			},
			user: users[userbilltoshipto],
			sc:   http.StatusOK,
		},
		"contact_patch": {
			postdata: shared.Contact{
				FirstName: "no firstname",
				LastName:  "lastname",
			},
			user: users[contactpatch],
			sc:   http.StatusOK,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%s/user/%s/contact", "localhost", "3000", tc.user.UUID),
				contactToReader(&tc.postdata))
			require.Nil(t, err)

			resp, err := (&http.Client{}).Do(req)
			require.Nil(t, err)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, "body: '%s'", body)
		})
	}
}
