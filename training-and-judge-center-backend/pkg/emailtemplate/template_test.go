package emailtemplate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

func TestWrap_PlacesTheTitleAndTheContentInsideTheLayout(t *testing.T) {
	html := emailtemplate.Wrap("Verify your email", "<p>Hello Ana</p>")

	assert.Contains(t, html, "<title>Verify your email</title>")
	assert.Contains(t, html, "<p>Hello Ana</p>")
	assert.Contains(t, html, "Training Center")
	assert.True(t, len(html) > 0 && html[:15] == "<!DOCTYPE html>")
	assert.Contains(t, html, `width="100%"`, "percent signs in the template are literal, not format verbs")
}

func TestWrap_DoesNotEscapeTheContent(t *testing.T) {
	html := emailtemplate.Wrap("t", `<a href="https://example.com">link</a>`)

	assert.Contains(t, html, `<a href="https://example.com">link</a>`)
}

func TestCodeBlock_ShowsTheCodeProminently(t *testing.T) {
	block := emailtemplate.CodeBlock("123456")

	assert.Contains(t, block, ">123456</span>")
	assert.Contains(t, block, "letter-spacing:8px")
}
