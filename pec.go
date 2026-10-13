package pec

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"time"
)

// Parse reads a raw RFC 822 / MIME message from r and returns a structured Message.
func Parse(r io.Reader) (*Message, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data)
}

// ParseBytes parses a raw message from b.
func ParseBytes(b []byte) (*Message, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	headers := extractPECHeaders(msg.Header)
	parts, err := walkMessage(msg)
	if err != nil {
		return nil, err
	}

	var dc *DatiCert
	if len(parts.DatiCertRaw) > 0 {
		dc, err = parseDatiCert(parts.DatiCertRaw)
		if err != nil {
			return nil, err
		}
	}

	kind := classify(headers, dc, headers.Subject)

	out := &Message{
		Kind:         kind,
		Subject:      headers.Subject,
		Headers:      headers,
		DatiCert:     dc,
		HasSignature: parts.HasSignature,
	}

	// Clean subject: prefer daticert <oggetto>.
	if dc != nil && dc.Oggetto != "" {
		out.CleanSubject = dc.Oggetto
	} else {
		out.CleanSubject = CleanSubject(headers.Subject)
	}

	// Message-ID / reference ID.
	out.MessageID = firstNonEmpty(headers.MessageID)
	if dc != nil && dc.Identificativo != "" {
		// Prefer certified identificativo when present (may lack <>).
		out.MessageID = normalizeMessageID(dc.Identificativo)
	} else {
		out.MessageID = normalizeMessageID(out.MessageID)
	}
	out.RefMessageID = firstNonEmpty(headers.RiferimentoMessageID)
	if out.RefMessageID == "" && dc != nil {
		out.RefMessageID = dc.MsgID
	}
	out.RefMessageID = normalizeMessageID(out.RefMessageID)

	// Sender.
	out.From = resolveFrom(headers, dc)

	// Date.
	out.Date = resolveDate(headers, dc)

	// Body and attachments: from inner postacert for transport/anomaly when present.
	useInner := (kind == KindTransport || kind == KindAnomaly) && len(parts.PostacertRaw) > 0
	if useInner {
		innerMsg, err := mail.ReadMessage(bytes.NewReader(parts.PostacertRaw))
		if err != nil {
			return nil, fmt.Errorf("postacert.eml: %w", err)
		}
		innerParts, err := walkMessage(innerMsg)
		if err != nil {
			return nil, fmt.Errorf("postacert.eml walk: %w", err)
		}
		out.BodyRaw, out.BodyMarkdown = pickBody(innerParts.PlainBodies, innerParts.HTMLBodies)
		out.Attachments = filterUserAttachments(innerParts.Attachments)
		// Anomaly envelopes often lack daticert; use the wrapped message's From.
		if kind == KindAnomaly && (dc == nil || dc.Mittente == "") {
			if f := addressOrRaw(innerMsg.Header.Get("From")); f != "" {
				out.From = f
			}
			if out.CleanSubject == CleanSubject(headers.Subject) || out.CleanSubject == "" {
				if subj := decodeRFC2047(innerMsg.Header.Get("Subject")); subj != "" {
					out.CleanSubject = subj
				}
			}
		}
	} else {
		out.BodyRaw, out.BodyMarkdown = pickBody(parts.PlainBodies, parts.HTMLBodies)
		out.Attachments = filterUserAttachments(parts.Attachments)
		// For receipts with postacert (completa/breve), merge inner attachments and .hash files.
		if len(parts.PostacertRaw) > 0 && kind == KindReceipt {
			innerMsg, err := mail.ReadMessage(bytes.NewReader(parts.PostacertRaw))
			if err == nil {
				if innerParts, err := walkMessage(innerMsg); err == nil {
					innerAtts := filterUserAttachments(innerParts.Attachments)
					out.Attachments = append(out.Attachments, innerAtts...)
					for k, v := range innerParts.HashChecksums {
						parts.HashChecksums[k] = v
					}
				}
			}
		}
	}

	if kind.IsService() {
		out.Receipt = buildReceiptInfo(headers, dc, parts.HashChecksums, kind)
	}

	return out, nil
}

// ParseFile opens path and parses the message.
func ParseFile(path string) (*Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func buildReceiptInfo(h PECHeaders, dc *DatiCert, hashes map[string]string, kind Kind) *ReceiptInfo {
	ri := &ReceiptInfo{
		Tipo:          receiptTipo(h, dc, kind),
		HashChecksums: hashes,
	}
	if dc != nil {
		ri.Errore = dc.Errore
		ri.ErroreEsteso = dc.ErroreEsteso
		ri.Consegna = dc.Consegna
		ri.Ricezione = append([]string(nil), dc.Ricezione...)
		ri.GestoreEmittente = dc.GestoreEmittente
		ri.RicevutaTipo = dc.RicevutaTipo
		ri.Identificativo = dc.Identificativo
		ri.MsgID = dc.MsgID
		ri.Data = dc.Data
	}
	if ri.Errore == "" && h.VerificaSicurezza != "" {
		ri.Errore = h.VerificaSicurezza
	}
	return ri
}

func resolveFrom(h PECHeaders, dc *DatiCert) string {
	if dc != nil && dc.Mittente != "" {
		return dc.Mittente
	}
	if h.Mittente != "" {
		return h.Mittente
	}
	if h.ReplyTo != "" {
		return addressOrRaw(h.ReplyTo)
	}
	return addressOrRaw(h.From)
}

func addressOrRaw(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return s
	}
	return addr.Address
}

func resolveDate(h PECHeaders, dc *DatiCert) time.Time {
	if dc != nil && !dc.Data.IsZero() {
		return dc.Data
	}
	if h.Date == "" {
		return time.Time{}
	}
	t, err := mail.ParseDate(h.Date)
	if err != nil {
		return time.Time{}
	}
	return t
}

func normalizeMessageID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	// Keep angle brackets if present; add them only when clearly an addr-spec without them.
	if strings.HasPrefix(id, "<") && strings.HasSuffix(id, ">") {
		return id
	}
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
