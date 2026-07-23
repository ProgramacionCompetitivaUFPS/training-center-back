package group

import (
	"context"
	"fmt"
	"html"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

// groupName is user-controlled (set by whoever created the group), so it is
// HTML-escaped before interpolation — emailtemplate.Wrap does not sanitize
// its content argument.
func sendInvitationEmail(
	ctx context.Context,
	emailSender appshared.EmailSender,
	frontendBaseURL string,
	groupName string,
	inv *domainGroup.GroupInvitation,
	invitee *UserDisplay,
) error {
	safeGroupName := html.EscapeString(groupName)
	acceptURL := fmt.Sprintf("%s/groups/%s/invitations/%s/accept", frontendBaseURL, inv.GroupID(), inv.ID())

	htmlContent := fmt.Sprintf(
		`<p style="margin:0 0 8px;">You've been invited to join <strong>%s</strong> on Training Center.</p>`+
			`<p style="margin:0 0 16px;">Click the button below to review and accept the invitation:</p>`+
			`<p style="text-align:center;margin:24px 0;"><a href="%s" style="background:#e11d48;color:#ffffff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600;">Accept invitation</a></p>`+
			`<p style="margin:0;color:#64748b;font-size:14px;">This invitation expires in 72 hours.</p>`,
		safeGroupName, acceptURL,
	)

	err := emailSender.Send(ctx, appshared.EmailMessage{
		To:       invitee.Email,
		Subject:  fmt.Sprintf("You've been invited to join %s", groupName),
		Body:     fmt.Sprintf("You've been invited to join %s on Training Center.\nAccept here: %s", groupName, acceptURL),
		HTMLBody: emailtemplate.Wrap("Group Invitation", htmlContent),
	})
	if err != nil {
		return apperror.NewServiceUnavailable(ErrCodeEmailDeliveryFailed, "failed to send invitation email")
	}
	return nil
}
