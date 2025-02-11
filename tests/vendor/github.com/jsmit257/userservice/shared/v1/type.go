package shared

import (
	"time"
)

type (
	UUID        string
	Password    string
	Email       string
	Cell        string
	CID         string
	CTXKey      string
	CustomError error

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
		UUID         UUID       `json:"id" mysql:"uuid"`
		Name         string     `json:"username" mysql:"name"`
		Pass         Password   `json:"password,omitempty" mysql:"password"`
		Salt         string     `json:"-" mysql:"salt"`
		LoginSuccess *time.Time `json:"login_success,omitempty" mysql:"loginsuccess"`
		LoginFailure *time.Time `json:"login_failure,omitempty" mysql:"loginfailure"`
		FailureCount uint8      `json:"failure_count,omitempty" mysql:"failurecount"`
		MTime        time.Time  `json:"mtime"`
		CTime        time.Time  `json:"ctime"`
	}

	Contact struct {
		FirstName string    `json:"first_name,omitempty"`
		LastName  string    `json:"last_name,omitempty"`
		BillTo    *Address  `json:"bill_to,omitempty"`
		ShipTo    *Address  `json:"ship_to,omitempty"`
		Phone     *string   `json:"phone,omitempty"`
		MTime     time.Time `json:"mtime"`
		CTime     time.Time `json:"ctime"`
	}

	User struct {
		UUID    UUID       `json:"id" mysql:"uuid"`
		Name    string     `json:"username" mysql:"name"`
		Contact *Contact   `json:"contact,omitempty"`
		Email   *Email     `json:"email,omitempty"`
		Cell    *Cell      `json:"cell,omitempty"`
		MTime   time.Time  `json:"mtime" mysql:"mtime"`
		CTime   time.Time  `json:"ctime" mysql:"ctime"`
		DTime   *time.Time `json:"dtime,omitempty" mysql:"dtime"`
	}
)
