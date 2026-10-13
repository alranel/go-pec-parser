package pec

import "time"

// Kind classifies a parsed message.
type Kind int

const (
	// KindOrdinary is a non-PEC message.
	KindOrdinary Kind = iota
	// KindTransport is a certified transport envelope (X-Trasporto: posta-certificata).
	KindTransport
	// KindReceipt is a PEC receipt or notice (accettazione, consegna, errori, …).
	KindReceipt
	// KindAnomaly is an anomaly wrapper (X-Trasporto: errore).
	KindAnomaly
)

func (k Kind) String() string {
	switch k {
	case KindTransport:
		return "transport"
	case KindReceipt:
		return "receipt"
	case KindAnomaly:
		return "anomaly"
	default:
		return "ordinary"
	}
}

// IsService reports whether the message is a PEC system message
// (receipt or anomaly), as opposed to certified content or ordinary mail.
func (k Kind) IsService() bool {
	return k == KindReceipt || k == KindAnomaly
}

// Message is the result of parsing a raw e-mail that may be a PEC message.
type Message struct {
	Kind         Kind
	Subject      string // raw Subject header
	CleanSubject string // original subject without PEC prefixes
	From         string // preferred sender (daticert mittente when present)
	Date         time.Time
	MessageID    string // envelope Message-ID / PEC identificativo
	RefMessageID string // original client Message-ID (X-Riferimento-Message-ID / msgid)

	BodyRaw      string // preferred text/plain (or HTML if that is all that is available)
	BodyMarkdown string // HTML converted to Markdown, or plain text normalized
	Attachments  []Attachment

	Receipt  *ReceiptInfo // non-nil for service messages (receipts and anomalies)
	Headers  PECHeaders
	DatiCert *DatiCert // parsed daticert.xml when present

	HasSignature bool // true if an S/MIME signature part was found (not verified)
}

// Attachment is a non-structural MIME part (user file, not daticert/postacert/smime).
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
	// HashSHA1 is set when this entry represents a brief-receipt .hash checksum
	// (40-char hex SHA-1 of the original attachment content).
	HashSHA1 string
}

// PECHeaders holds PEC-specific header fields from the outer envelope.
type PECHeaders struct {
	Trasporto            string // posta-certificata | errore
	Ricevuta             string // accettazione, avvenuta-consegna, …
	TipoRicevuta         string // completa | breve | sintetica
	RiferimentoMessageID string
	VerificaSicurezza    string // e.g. errore on virus-related notices
	Mittente             string // X-Mittente (virus notices)
	MessageID            string
	From                 string
	ReplyTo              string
	Subject              string
	Date                 string
}

// Destinatario is a recipient listed in daticert.xml.
type Destinatario struct {
	Address string
	Tipo    string // certificato | esterno
}

// DatiCert is the parsed content of daticert.xml (certification data).
type DatiCert struct {
	Tipo             string // postacert@tipo
	Errore           string // postacert@errore (nessuno, no-dest, virus, …)
	Mittente         string
	Destinatari      []Destinatario
	Risposte         string
	Oggetto          string
	GestoreEmittente string
	Data             time.Time
	Identificativo   string
	MsgID            string // original client Message-ID (typically with <>)
	RicevutaTipo     string // completa | breve | sintetica
	Consegna         string
	Ricezione        []string
	ErroreEsteso     string
}

// ReceiptInfo aggregates service-message metadata useful for event handling
// and legal identification of the certified event.
type ReceiptInfo struct {
	Tipo             string // accettazione, avvenuta-consegna, …
	Errore           string
	ErroreEsteso     string
	Consegna         string
	Ricezione        []string
	GestoreEmittente string
	RicevutaTipo     string
	Identificativo   string
	MsgID            string
	Data             time.Time
	// HashChecksums maps original attachment filenames (without .hash) to SHA-1 hex.
	HashChecksums map[string]string
}
