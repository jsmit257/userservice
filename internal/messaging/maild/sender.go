package maild

import (
	"fmt"

	"github.com/go-gomail/gomail"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
)

type (
	Sender interface {
		Send(m *gomail.Message) error
		Close()
	}

	sender chan *gomail.Message

	testSender struct {
		l *logrus.Entry
	}
)

func NewSender(cfg *config.Config, log *logrus.Entry) (Sender, error) {
	log = log.WithField("pkg", "maild")
	result := make(sender, 10)

	if cfg.EmailTestMode {
		log.Info("mail relay daemon started with dummy mailer")
		return &testSender{log}, nil
	}

	d := gomail.NewDialer(
		cfg.MaildHost,
		int(cfg.MaildPort),
		cfg.MaildUser,
		cfg.MaildPass)

	// ping
	if s, err := d.Dial(); err != nil {
		return nil, err
	} else if err = s.Close(); err != nil {
		return nil, err
	}

	go func() {
		for msg := range result {
			l := log.WithFields(logrus.Fields{
				"rx":      msg.GetHeader("To"),
				"subject": msg.GetHeader("Subject"),
			})

			msg.SetHeader("From", cfg.MaildSender)

			if err := d.DialAndSend(msg); err != nil {
				l.WithError(err).Error("failed to send message")
			} else {
				l.Info("sent message")
			}
		}
		log.Info("mail daemon channel closed")
	}()

	log.Info("mail relay daemon started")

	return result, nil
}

func (s sender) Send(m *gomail.Message) (err error) {
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

func (s *testSender) Send(msg *gomail.Message) error {
	s.l.WithField("msg", msg).Info("sending message")
	return nil
}

func (s *testSender) Close() {
	s.l.Info("closing maild")
}
