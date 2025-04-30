package maild

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/internal/config"
)

func Test_NewSender(t *testing.T) {
	t.Skip() // stand up a smtp image to handle this? or just save it for system tests?
	t.Parallel()

	cfg := &config.Config{
		MaildHost: os.Getenv("MAILD_RELAY_HOST"),
		MaildPort: func(s string) uint16 {
			i, err := strconv.Atoi(s)
			if err != nil {
				panic(err)
			}
			return uint16(i)
		}(os.Getenv("MAILD_RELAY_PORT")),
		MaildUser: os.Getenv("MAILD_RELAY_USER"),
		MaildPass: os.Getenv("MAILD_RELAY_PASS"),
	}

	sender, err := NewSender(cfg, logrus.WithField("app", "maild-test"))
	require.Nil(t, err)

	err = sender.Send(&gomail.Message{})
	require.Nil(t, err)

	sender.Close()

	err = sender.Send(&gomail.Message{})
	require.NotNil(t, err)
	require.Equal(t, "send on closed channel", err.Error())
}

func Test_testSender(t *testing.T) {
	t.Parallel()

	s, err := NewSender(&config.Config{EmailTestMode: true}, logrus.WithField("test", "Test_testSender"))
	require.Nil(t, err)
	err = s.Send(nil)
	require.Nil(t, err)
	s.Close()
}

func Test_Send(t *testing.T) {
	t.Parallel()

	var err error
	s := make(sender)
	m := gomail.NewMessage()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Send(m)
	}()
	require.Equal(t, m, <-s, "happy_path")
	wg.Wait()
	require.Nil(t, err, "happy_path")

	err = s.Send(nil)
	require.Nil(t, err, "sending_nil")
	select {
	case m, ok := <-s:
		if ok {
			require.Failf(t, "sending_nil", "channel should not have a message on it: %#v", m)
		} else {
			require.Fail(t, "sending_nil", "channel should not be closed")
		}
	case <-time.After(time.Millisecond):
		t.Logf("sending_nil: %s", "everything OK")
	}

	s.Close()
	require.Nil(t, err, "closing_channel")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Send(m)
	}()
	wg.Wait()
	require.NotNil(t, err, "sending closed channel")
	require.Equal(t, "send on closed channel", err.Error(), "sending closed channel") // ???
}

func Test_Close(t *testing.T) {
	t.Parallel()

	s := make(sender)
	s.Close()

	defer func() {
		require.NotNil(t, recover())
	}()

	s <- gomail.NewMessage()
	require.Fail(t, "should've paniced")

}
