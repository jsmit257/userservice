package shared

import (
	"time"
)

type (
	UUID string
	CID  string

	Address struct {
		UUID    UUID      `json:"id" mysql:"uuid"`
		Street1 string    `json:"street1"`
		Street2 string    `json:"street2,omitempty"`
		City    string    `json:"city"`
		State   string    `json:"state"`
		Country string    `json:"country"`
		Zip     string    `json:"zip"`
		MTime   time.Time `json:"mtime"`
		CTime   time.Time `json:"ctime"`
	}

	BasicAuth struct {
		UUID UUID   `json:"id" mysql:"uuid"`
		Name string `json:"username" mysql:"name"`
		Pass string `json:"password" mysql:"password"`
		Salt string `json:"-" mysql:"salt"`
	}

	Contact struct {
		UUID      UUID      `json:"id" mysql:"uuid"`
		FirstName string    `json:"first_name,omitempty"`
		LastName  string    `json:"last_name,omitempty"`
		BillTo    *Address  `json:"bill_to,omitempty"`
		ShipTo    *Address  `json:"ship_to,omitempty"`
		MTime     time.Time `json:"mtime"`
		CTime     time.Time `json:"ctime"`
	}

	User struct {
		UUID         UUID       `json:"id" mysql:"uuid"`
		Name         string     `json:"username" mysql:"name"`
		Contact      *Contact   `json:"contact,omitempty"`
		MTime        time.Time  `json:"mtime" mysql:"mtime"`
		CTime        time.Time  `json:"ctime" mysql:"ctime"`
		DTime        *time.Time `json:"dtime,omitempty" mysql:"dtime"`
		LoginSuccess *time.Time `json:"login_success,omitempty" mysql:"mtime"`
		FailureCount uint8      `json:"failure_count,omitempty" mysql:"failure_count"`
	}
)
