package email

import (
	"context"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/training-judge-center/backend/internal/application/shared"
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

func (s *SMTPSender) Send(ctx context.Context, msg shared.EmailMessage) error {
	toAddr, err := mail.ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	safeSubject := mime.QEncoding.Encode("utf-8", strings.ReplaceAll(msg.Subject, "\r\n", " "))

	var auth smtp.Auth
	if s.username != "" || s.password != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	raw := []byte("To: " + toAddr.String() + "\r\n" +
		"From: " + s.from + "\r\n" +
		"Subject: " + safeSubject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" + msg.Body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	if err := smtp.SendMail(addr, auth, s.from, []string{toAddr.Address}, raw); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
