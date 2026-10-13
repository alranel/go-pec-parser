package pec

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
)

// mimeParts collects interesting pieces while walking a MIME tree.
type mimeParts struct {
	DatiCertRaw   []byte
	PostacertRaw  []byte
	HasSignature  bool
	PlainBodies   []string
	HTMLBodies    []string
	Attachments   []Attachment
	HashChecksums map[string]string // filename without .hash → SHA-1 hex
}

func walkMessage(msg *mail.Message) (*mimeParts, error) {
	parts := &mimeParts{HashChecksums: make(map[string]string)}
	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}
	if err := walkPart(ct, msg.Header, msg.Body, parts); err != nil {
		return nil, err
	}
	return parts, nil
}

func walkPart(contentType string, header mail.Header, body io.Reader, parts *mimeParts) error {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "application/octet-stream"
		params = map[string]string{}
	}

	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("multipart without boundary")
		}
		mr := multipart.NewReader(body, boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("multipart: %w", err)
			}
			partCT := p.Header.Get("Content-Type")
			if partCT == "" {
				partCT = "text/plain"
			}
			partHeader := mail.Header(p.Header)
			if err := walkPart(partCT, partHeader, p, parts); err != nil {
				return err
			}
		}
		return nil

	case mediaType == "message/rfc822":
		data, err := decodeBody(header, body)
		if err != nil {
			return err
		}
		filename := partFilename(header, params)
		// Structural postacert.eml (or unnamed rfc822 inside a PEC envelope).
		if isPostacertName(filename) || filename == "" {
			if isPostacertName(filename) || parts.PostacertRaw == nil {
				parts.PostacertRaw = data
			}
			return nil
		}
		parts.Attachments = append(parts.Attachments, Attachment{
			Filename:    filename,
			ContentType: mediaType,
			Data:        data,
		})
		return nil

	default:
		data, err := decodeBody(header, body)
		if err != nil {
			return err
		}
		filename := partFilename(header, params)
		disp, dispParams, _ := mime.ParseMediaType(header.Get("Content-Disposition"))

		if isDaticert(mediaType, filename) {
			parts.DatiCertRaw = data
			return nil
		}
		if isSignature(mediaType, filename) {
			parts.HasSignature = true
			return nil
		}
		if isPostacertName(filename) {
			parts.PostacertRaw = data
			return nil
		}

		// Brief-receipt SHA-1 hash files
		if strings.HasSuffix(strings.ToLower(filename), ".hash") &&
			(mediaType == "text/plain" || mediaType == "application/octet-stream" || mediaType == "") {
			key := strings.TrimSuffix(filename, ".hash")
			key = strings.TrimSuffix(key, ".HASH")
			hash := strings.TrimSpace(string(data))
			parts.HashChecksums[key] = hash
			parts.Attachments = append(parts.Attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				Data:        data,
				HashSHA1:    hash,
			})
			return nil
		}

		isAttach := strings.EqualFold(disp, "attachment") ||
			(dispParams["filename"] != "" && !strings.EqualFold(disp, "inline"))

		switch mediaType {
		case "text/plain":
			if isAttach && filename != "" {
				parts.Attachments = append(parts.Attachments, Attachment{
					Filename:    filename,
					ContentType: mediaType,
					Data:        data,
				})
			} else {
				parts.PlainBodies = append(parts.PlainBodies, decodeCharset(string(data), params["charset"]))
			}
			return nil
		case "text/html":
			if isAttach && filename != "" {
				parts.Attachments = append(parts.Attachments, Attachment{
					Filename:    filename,
					ContentType: mediaType,
					Data:        data,
				})
			} else {
				parts.HTMLBodies = append(parts.HTMLBodies, decodeCharset(string(data), params["charset"]))
			}
			return nil
		default:
			if isStructuralName(filename) {
				return nil
			}
			if filename == "" && !isAttach {
				// skip unnamed non-text inline parts
				return nil
			}
			if filename == "" {
				filename = "attachment"
			}
			parts.Attachments = append(parts.Attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				Data:        data,
			})
			return nil
		}
	}
}

func isDaticert(mediaType, filename string) bool {
	if strings.EqualFold(filename, "daticert.xml") || strings.EqualFold(filename, "certdata.xml") {
		return true
	}
	return mediaType == "application/xml" && strings.Contains(strings.ToLower(filename), "daticert")
}

func isSignature(mediaType, filename string) bool {
	switch mediaType {
	case "application/pkcs7-signature", "application/x-pkcs7-signature",
		"application/pkcs7-mime", "application/x-pkcs7-mime":
		return true
	}
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".p7s") || strings.HasSuffix(lower, ".p7m") ||
		strings.EqualFold(filename, "smime.p7s") || strings.EqualFold(filename, "smime.p7m")
}

func isPostacertName(filename string) bool {
	lower := strings.ToLower(filename)
	return lower == "postacert.eml" || lower == "postacert"
}

func isStructuralName(filename string) bool {
	if isPostacertName(filename) || isDaticert("", filename) || isSignature("", filename) {
		return true
	}
	lower := strings.ToLower(filename)
	return lower == "daticert.xml" || lower == "certdata.xml"
}

func partFilename(header mail.Header, ctParams map[string]string) string {
	if _, params, err := mime.ParseMediaType(header.Get("Content-Disposition")); err == nil {
		if fn := params["filename"]; fn != "" {
			return decodeRFC2047(fn)
		}
	}
	if fn := ctParams["name"]; fn != "" {
		return decodeRFC2047(fn)
	}
	return ""
}

func decodeBody(header mail.Header, body io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	enc := strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding")))
	switch enc {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(stripWhitespace(string(raw)))
		if err != nil {
			// try raw URL-safe / lenient
			decoded, err = base64.RawStdEncoding.DecodeString(stripWhitespace(string(raw)))
			if err != nil {
				return raw, nil // fall back to raw bytes
			}
		}
		return decoded, nil
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
		if err != nil {
			return raw, nil
		}
		return decoded, nil
	default:
		return raw, nil
	}
}

func stripWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' && r != '\n' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var wordDecoder = &mime.WordDecoder{}

func decodeRFC2047(s string) string {
	decoded, err := wordDecoder.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

// decodeCharset currently assumes UTF-8 / ASCII for body text.
// Charset conversion can be extended with golang.org/x/text if needed.
func decodeCharset(s, charset string) string {
	_ = charset
	return s
}

func extractPECHeaders(h mail.Header) PECHeaders {
	return PECHeaders{
		Trasporto:            h.Get("X-Trasporto"),
		Ricevuta:             h.Get("X-Ricevuta"),
		TipoRicevuta:         h.Get("X-TipoRicevuta"),
		RiferimentoMessageID: h.Get("X-Riferimento-Message-ID"),
		VerificaSicurezza:    h.Get("X-VerificaSicurezza"),
		Mittente:             h.Get("X-Mittente"),
		MessageID:            h.Get("Message-ID"),
		From:                 h.Get("From"),
		ReplyTo:              h.Get("Reply-To"),
		Subject:              decodeRFC2047(h.Get("Subject")),
		Date:                 h.Get("Date"),
	}
}
