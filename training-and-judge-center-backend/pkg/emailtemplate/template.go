package emailtemplate

import "fmt"

// Wrap returns a complete HTML email using the Training Center design tokens.
// title appears in the <title> tag; content is raw HTML injected into the body card.
func Wrap(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f8fafc;font-family:Inter,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f8fafc;padding:40px 16px;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border:1px solid #e2e8f0;border-radius:12px;overflow:hidden;max-width:560px;width:100%%;">
        <tr>
          <td style="background:#e11d48;padding:24px 32px;">
            <span style="color:#f9fafb;font-size:20px;font-weight:700;letter-spacing:-0.01em;">Training Center</span>
          </td>
        </tr>
        <tr>
          <td style="padding:32px;color:#0f172a;font-size:16px;line-height:1.5;">
            %s
          </td>
        </tr>
        <tr>
          <td style="background:#f8fafc;padding:16px 32px;border-top:1px solid #e2e8f0;text-align:center;">
            <span style="color:#64748b;font-size:12px;">Training Center &middot; UFPS</span>
          </td>
        </tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, title, content)
}

// CodeBlock returns a styled HTML block displaying a verification code prominently.
func CodeBlock(code string) string {
	return fmt.Sprintf(
		`<div style="text-align:center;margin:24px 0;">
  <span style="display:inline-block;background:#fff1f2;border:2px dashed #e11d48;border-radius:12px;padding:16px 32px;font-size:32px;font-weight:700;letter-spacing:8px;color:#e11d48;font-family:'JetBrains Mono',SFMono-Regular,Menlo,Monaco,Consolas,'Courier New',monospace;">%s</span>
</div>`, code)
}
