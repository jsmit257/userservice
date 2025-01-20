package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSqls(t *testing.T) {
	t.Parallel()

	err := os.Chdir("../..")
	require.Nil(t, err)

	tcs := map[string]struct {
		vendor         string
		result, actual Sqls
		err            bool
	}{
		"happy_path": {
			vendor: "mysql",
			result: Sqls{
				"address": map[string]string{
					"insert":     "insert into  addresses( uuid, street1, street2, city, state, country, zip, mtime, ctime) values  (?, ?, ?, ?, ?, ?, ?, ?, ?)",
					"select":     "select  uuid, street1, street2, city, state, country, zip, mtime, ctime from  addresses where  uuid = ?",
					"select-all": "select  uuid, street1, street2, city, state, country, zip, mtime, ctime from  addresses",
					"update":     "update  addresses set  street1 = ?, street2 = ?, city = ?, state = ?, country = ?, zip = ?, mtime = ? where  uuid = ?",
				},
				"basic-auth": map[string]string{
					"select": "select  uuid, name, password, salt, loginsuccess, loginfailure, failurecount, mtime, ctime from  users where  uuid = coalesce(?, uuid) and  name = coalesce(?, name)",
					"update": "update  users set  password = ?, salt = ?, loginsuccess = ?, loginfailure = ?, failurecount = ?, mtime = current_timestamp where  uuid = ?",
				},
				"contact": map[string]string{
					"insert": "insert into  contacts( uuid, firstname, lastname, billto_uuid, shipto_uuid, mtime, ctime) select  uuid, ?, ?, ?, ?, ?, ? from users where uuid = ?",
					"select": "select  firstname, lastname, billto_uuid, shipto_uuid, mtime, ctime from  contacts where  uuid = ?",
					"update": "update  contacts set  firstname = ?, lastname = ?, billto_uuid = ?, shipto_uuid = ?, mtime = ? where  uuid = ?",
				},
				"user": map[string]string{
					"delete":     "update users set dtime = ? where uuid = ?",
					"insert":     "insert into  users(uuid, name, password, salt, mtime, ctime) values  (?, ?, ?, ?, ?, ?)",
					"select":     "select  uuid, name, email, cell, mtime, ctime, dtime from  users where  uuid = ?",
					"select-all": "select  uuid, name, mtime, ctime, dtime from  users",
					"update":     "update  users set  name = ?, mtime = ? where  uuid = ?",
				},
			},
		},
		"sad_path": {
			vendor: "unknown",
			err:    true,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := NewSqls(tc.vendor)

			if tc.err {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err, func(s string, e error) string { return s }(os.Getwd()))
			}
			require.Equal(t, tc.result, result)
		})
	}
}
