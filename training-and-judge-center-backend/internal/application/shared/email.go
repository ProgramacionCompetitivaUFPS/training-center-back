package shared

import "context"

type EmailMessage struct {
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
