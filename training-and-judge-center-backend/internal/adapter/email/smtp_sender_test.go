package email_test

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/adapter/email"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type capturedMail struct {
	from, rcpt, data, auth string
}

// fakeSMTP is a minimal SMTP server: enough of the protocol for net/smtp.SendMail.
type fakeSMTP struct {
	addr      string
	listener  net.Listener
	mu        sync.Mutex
	mail      capturedMail
	offerAuth bool
}

func startFakeSMTP(t *testing.T, offerAuth bool) *fakeSMTP {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeSMTP{addr: l.Addr().String(), listener: l, offerAuth: offerAuth}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	return s
}

func (s *fakeSMTP) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
	write("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			if s.offerAuth {
				write("250-fake")
				write("250 AUTH PLAIN")
			} else {
				write("250 fake")
			}
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			s.mu.Lock()
			s.mail.auth = line
			s.mu.Unlock()
			write("235 ok")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.mu.Lock()
			s.mail.from = strings.Trim(line[len("MAIL FROM:"):], "<> ")
			s.mu.Unlock()
			write("250 ok")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.mu.Lock()
			s.mail.rcpt = strings.Trim(line[len("RCPT TO:"):], "<> ")
			s.mu.Unlock()
			write("250 ok")
		case upper == "DATA":
			write("354 go ahead")
			var data strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil || l == ".\r\n" {
					break
				}
				data.WriteString(l)
			}
			s.mu.Lock()
			s.mail.data = data.String()
			s.mu.Unlock()
			write("250 queued")
		case upper == "QUIT":
			write("221 bye")
			return
		default:
			write("250 ok")
		}
	}
}

func (s *fakeSMTP) received() capturedMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mail
}

func senderFor(s *fakeSMTP, username, password, from string) *email.SMTPSender {
	host, port, _ := net.SplitHostPort(s.addr)
	return email.NewSMTPSender(host, port, username, password, from)
}

func TestSMTPSender_Send_PlainTextMessage(t *testing.T) {
	server := startFakeSMTP(t, false)
	sender := senderFor(server, "", "", "Training Center <noreply@trainingcenter.com>")

	err := sender.Send(context.Background(), shared.EmailMessage{To: "ana@example.com", Subject: "Your code", Body: "123456"})

	require.NoError(t, err)
	got := server.received()
	assert.Equal(t, "noreply@trainingcenter.com", got.from)
	assert.Equal(t, "ana@example.com", got.rcpt)
	assert.Contains(t, got.data, "To: <ana@example.com>")
	assert.Contains(t, got.data, "From: Training Center <noreply@trainingcenter.com>")
	assert.Contains(t, got.data, "Subject: Your code")
	assert.Contains(t, got.data, `Content-Type: text/plain; charset="utf-8"`)
	assert.True(t, strings.HasSuffix(strings.TrimRight(got.data, "\r\n"), "123456"))
	assert.Empty(t, got.auth, "no credentials configured, no AUTH")
}

func TestSMTPSender_Send_HTMLMessageIsMultipartWithBothParts(t *testing.T) {
	server := startFakeSMTP(t, false)
	sender := senderFor(server, "", "", "noreply@trainingcenter.com")

	err := sender.Send(context.Background(), shared.EmailMessage{To: "ana@example.com", Subject: "Hi", Body: "plain body", HTMLBody: "<p>html body</p>"})

	require.NoError(t, err)
	data := server.received().data
	assert.Contains(t, data, "Content-Type: multipart/alternative")
	assert.Contains(t, data, `Content-Type: text/plain; charset="utf-8"`+"\r\n\r\nplain body")
	assert.Contains(t, data, `Content-Type: text/html; charset="utf-8"`+"\r\n\r\n<p>html body</p>")
}

func TestSMTPSender_Send_SubjectCannotInjectHeadersAndIsEncodedWhenNotASCII(t *testing.T) {
	server := startFakeSMTP(t, false)
	sender := senderFor(server, "", "", "noreply@trainingcenter.com")

	require.NoError(t, sender.Send(context.Background(), shared.EmailMessage{
		To: "ana@example.com", Subject: "Hello\r\nBcc: attacker@evil.com", Body: "x",
	}))
	injected := server.received().data
	assert.NotContains(t, injected, "\r\nBcc:")
	assert.Contains(t, injected, "Subject: Hello Bcc: attacker@evil.com")

	require.NoError(t, sender.Send(context.Background(), shared.EmailMessage{To: "ana@example.com", Subject: "Verificación", Body: "x"}))
	assert.Contains(t, server.received().data, "Subject: =?utf-8?q?Verificaci=C3=B3n?=")
}

func TestSMTPSender_Send_AuthenticatesWhenCredentialsAreConfigured(t *testing.T) {
	server := startFakeSMTP(t, true)
	sender := senderFor(server, "mailer", "secret", "noreply@trainingcenter.com")

	err := sender.Send(context.Background(), shared.EmailMessage{To: "ana@example.com", Subject: "Hi", Body: "x"})

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(server.received().auth, "AUTH PLAIN "))
}

func TestSMTPSender_Send_InvalidAddressesAreInternalErrorsAndSendNothing(t *testing.T) {
	server := startFakeSMTP(t, false)
	valid := senderFor(server, "", "", "noreply@trainingcenter.com")
	badFrom := senderFor(server, "", "", "not an address")

	tests := []struct {
		name   string
		sender *email.SMTPSender
		to     string
	}{
		{"invalid recipient", valid, "not-an-email"},
		{"invalid configured sender", badFrom, "ana@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sender.Send(context.Background(), shared.EmailMessage{To: tt.to, Subject: "Hi", Body: "x"})

			var appErr *apperror.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, apperror.KindInternal, appErr.Kind)
			assert.Empty(t, server.received().rcpt)
		})
	}
}

func TestSMTPSender_Send_UnreachableServerIsAnInternalError(t *testing.T) {
	sender := email.NewSMTPSender("127.0.0.1", "1", "", "", "noreply@trainingcenter.com")

	err := sender.Send(context.Background(), shared.EmailMessage{To: "ana@example.com", Subject: "Hi", Body: "x"})

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}
