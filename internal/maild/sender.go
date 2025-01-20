package maild

import (
	"fmt"

	"github.com/go-gomail/gomail"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
)

type Sender interface {
	Send(m *gomail.Message) error
	Close()
}

type sender chan *gomail.Message

func NewSender(cfg *config.Config, log *logrus.Entry) (Sender, error) {
	log = log.WithField("pkg", "maild")
	result := make(sender, 10)

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

			if err := d.DialAndSend(msg); err != nil {
				l.WithError(err).Error("failed to send message")
			} else {
				l.Info("sent message")
			}
		}

	}()

	log.Info("mail relay daemon started")

	return result, nil
}

func (s sender) Send(m *gomail.Message) error {
	var err error
	defer func() {
		if maybe, ok := recover().(error); ok {
			err = maybe
		} else {
			err = fmt.Errorf("non-error in panic: %#v", err)
		}
	}()

	if m != nil {
		s <- m
	}

	return err
}

func (s sender) Close() {
	close(s)
}
