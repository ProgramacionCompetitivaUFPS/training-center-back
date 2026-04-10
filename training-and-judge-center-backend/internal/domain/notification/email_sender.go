package notification

import "context"

// EmailMessage groups the fields required to send an email.
// Using a struct instead of positional parameters makes the interface
// easier to extend in the future (e.g. CC, HTML body) without breaking callers.
type EmailMessage struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
