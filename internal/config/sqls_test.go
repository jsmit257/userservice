package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSqls(t *testing.T) {
	t.Skip()
	t.Parallel()

	tcs := map[string]struct {
		vendor string
		result Sqls
		err    bool
	}{
		"happy_path": {
			vendor: "mysql",
			result: Sqls{
				"address": map[string]string{
					"delete":     "update addresses set dtime = ? where uuid = ?",
					"insert":     "insert into  addresses( uuid, street1, street2, city, state, country, zip, mtime, ctime) values  (?, ?, ?, ?, ?, ?, ?, ?, ?)",
					"select":     "select  uuid, street1, street2, city, state, country, zip, mtime, ctime, dtime from  addresses where  uuid = ?",
					"select-all": "select  uuid, street1, street2, city, state, country, zip, mtime, ctime, dtime from  addresses",
					"update":     "update  addresses set  street1 = ?, street2 = ?, city = ?, state = ?, country = ?, zip = ?, mtime = ? where  uuid = ?",
				},
				"basic-auth": map[string]string{
					"select": "select  id, password, salt, loginsuccess, loginfailure, failurecount from  users where  uuid = ?",
					"update": "update  users set  password = ?, salt = ?, loginsuccess = ?, loginfailure = ?, failurecount = ?, mtime = current_timestamp where  uuid = ?",
				},
				"contact": map[string]string{
					"delete": "update contact set dtime = ? where uuid = ?",
					"insert": "insert into  contacts( uuid, firstname, lastname, billto_uuid, shipto_uuid, mtime, ctime) values  (?, ?, ?, ?, ?, ?, ?)",
					"update": "update  contacts set  uuid = ?, firstname = ?, lastname = ?, billto_uuid = ? shipto_uuid = ? mtime = ? where  uuid = ?",
				},
				"user": map[string]string{
					"delete":     "update users set dtime = ? where uuid = ?",
					"insert":     "insert into  users(uuid, name, password, salt, mtime, ctime) values  (?, ?, ?, ?, ?, ?)",
					"select":     "select  u.id, u.name, u.mtime, u.ctime, u.dtime, c.firstname, c.lastname, c.billto_uuid, c.shipto_uuid, c.mtime as contact_mtime, c.ctime as contact_ctime, c.dtime as contact_dtime from  users u left join  contacts c on  u.uuid = c.uuid where  uuid = ?",
					"select-all": "select  id, name, mtime, ctime, dtime, loginsuccess, loginfailure, failurecount from  users",
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
				require.Nil(t, err)
			}
			require.Equal(t, tc.result, result)
		})
	}
}
