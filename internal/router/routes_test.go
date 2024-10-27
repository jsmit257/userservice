package router

import (
	"fmt"
	"os"
	"testing"

	"github.com/jsmit257/userservice/internal/config"
)

type errReader string

func (er errReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("%s", er)
}

func Test_NewInstance(t *testing.T) {
	t.Parallel()

	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_PORT", "1234")
	os.Setenv("MYSQL_USER", "fake")
	os.Setenv("MYSQL_PASSWORD", "snakeoil")

	_ = NewInstance(&UserService{
		Addresser: &mockAddresser{},
		Contacter: &mockContacter{},
		Userer:    &mockUserer{},
	}, config.NewConfig(), nil)
}
