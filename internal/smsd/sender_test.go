package smsd

import (
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/jsmit257/userservice/internal/config"
)

func Test_Send(t *testing.T) {
	t.Parallel()

	var err error
	s := make(sender)
	m := &twilioApi.CreateMessageParams{}
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
	require.Equal(t, nil, err, "sending_closed_channel") // ???
}

func Test_Close(t *testing.T) {
	t.Parallel()

	s := make(sender)
	s.Close()

	defer func() {
		require.NotNil(t, recover())
	}()

	s <- &twilioApi.CreateMessageParams{}
	require.Fail(t, "should've paniced")

}

func Test_testSender(t *testing.T) {
	t.Parallel()

	s, err := NewSender(&config.Config{SmsTestMode: true}, logrus.WithField("test", "Test_testSender"))
	require.Nil(t, err)
	err = s.Send(nil)
	require.Nil(t, err)
	s.Close()
}
