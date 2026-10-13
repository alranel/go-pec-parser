package pec

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// pickBody chooses raw text and Markdown from collected plain/HTML parts.
// Prefer plain text for BodyRaw; use HTML→Markdown when HTML is available,
// otherwise normalize plain text as Markdown.
func pickBody(plain, html []string) (raw, markdown string) {
	plainJoined := joinBodies(plain)
	htmlJoined := joinBodies(html)

	if plainJoined != "" {
		raw = plainJoined
	} else {
		raw = htmlJoined
	}

	if htmlJoined != "" {
		converted, err := md.ConvertString(htmlJoined)
		if err == nil && strings.TrimSpace(converted) != "" {
			markdown = strings.TrimSpace(converted)
			return raw, markdown
		}
	}
	markdown = normalizePlainAsMarkdown(plainJoined)
	if markdown == "" && htmlJoined != "" {
		markdown = normalizePlainAsMarkdown(stripTagsRough(htmlJoined))
	}
	return raw, markdown
}

func joinBodies(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p)
	}
	return b.String()
}

func normalizePlainAsMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// stripTagsRough is a last-resort fallback when html-to-markdown fails.
func stripTagsRough(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// filterUserAttachments drops structural PEC parts and returns user-facing attachments
// plus any .hash checksum map already collected.
func filterUserAttachments(atts []Attachment) []Attachment {
	out := make([]Attachment, 0, len(atts))
	for _, a := range atts {
		if isStructuralName(a.Filename) || isSignature(a.ContentType, a.Filename) {
			continue
		}
		if isDaticert(a.ContentType, a.Filename) {
			continue
		}
		out = append(out, a)
	}
	return out
}
