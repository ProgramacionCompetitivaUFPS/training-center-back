package email

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/training-judge-center/backend/internal/domain/notification"
)

type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(host, port, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Send(ctx context.Context, msg notification.EmailMessage) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	raw := []byte("To: " + msg.To + "\r\n" +
		"From: " + s.from + "\r\n" +
		"Subject: " + msg.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" + msg.Body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	if err := smtp.SendMail(addr, auth, s.from, []string{msg.To}, raw); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
