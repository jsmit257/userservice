package smsd

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/jsmit257/userservice/internal/config"
)

type (
	Sender interface {
		Send(m *twilioApi.CreateMessageParams) error
		Close()
	}

	sender chan *twilioApi.CreateMessageParams

	testSender struct {
		l *logrus.Entry
	}
)

func NewSender(cfg *config.Config, log *logrus.Entry) (Sender, error) {
	log = log.WithField("pkg", "smsd")
	result := make(sender, 10)

	if cfg.SmsTestMode {
		log.Info("sms relay daemon started with dummy sender")
		return &testSender{log}, nil
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.SmsAccountID,
		Password: cfg.SmsAuthToken,
	})

	go func() {
		for msg := range result {
			l := log.WithField("rx", msg.To)

			resp, err := client.Api.CreateMessage(msg.
				SetFrom(cfg.SmsSender).
				SetTrafficType("").                       // XXX: A2P, Transactional, ???
				SetContentRetention("").                  // XXX: Limited?
				SetValidityPeriod(int(cfg.AuthnTimeout))) // XXX: does twilio use minutes?
			if err != nil {
				l.WithError(err).Error("sending SMS message")
			} else {
				s, _ := json.Marshal(resp)
				l.WithField("send-response", s).Info("sms sent")
			}
		}
		log.Info("sms daemon channel closed")
	}()

	log.Info("sms relay daemon started")

	return result, nil
}

func (s sender) Send(m *twilioApi.CreateMessageParams) (err error) {
	defer func() {
		if maybe, ok := recover().(error); ok {
			err = maybe
		} else if maybe != nil {
			err = fmt.Errorf("non-error in panic: %w", err)
		}
	}()

	if m != nil {
		s <- m
	}

	return
}

func (s sender) Close() {
	close(s)
}

func (s *testSender) Send(msg *twilioApi.CreateMessageParams) error {
	s.l.WithField("msg", msg).Info("sending message")
	return nil
}

func (s *testSender) Close() {
	s.l.Info("closing smsd")
}
