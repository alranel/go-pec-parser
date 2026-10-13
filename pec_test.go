package pec

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFixtures(t *testing.T) {
	tests := []struct {
		file         string
		kind         Kind
		cleanSubject string
		from         string
		refID        string
		receiptTipo  string
		hasAttach    string // filename expected in attachments (empty = none required)
		errore       string
		consegna     string
		hasSig       bool
	}{
		{
			file:         "transport.eml",
			kind:         KindTransport,
			cleanSubject: "Contratto firmato",
			from:         "alice@pec.example.it",
			refID:        "<client-id@mail.client>",
			hasAttach:    "contratto.pdf",
			hasSig:       true,
		},
		{
			file:         "accettazione.eml",
			kind:         KindReceipt,
			cleanSubject: "Contratto firmato",
			from:         "alice@pec.example.it",
			refID:        "<client-id@mail.client>",
			receiptTipo:  "accettazione",
			hasSig:       true,
		},
		{
			file:         "avvenuta-consegna.eml",
			kind:         KindReceipt,
			cleanSubject: "Contratto firmato",
			from:         "alice@pec.example.it",
			refID:        "<client-id@mail.client>",
			receiptTipo:  "avvenuta-consegna",
			consegna:     "bob@pec.altro.it",
		},
		{
			file:         "errore-consegna.eml",
			kind:         KindReceipt,
			cleanSubject: "Contratto firmato",
			from:         "alice@pec.example.it",
			refID:        "<client-id@mail.client>",
			receiptTipo:  "errore-consegna",
			errore:       "altro",
			consegna:     "bob@pec.altro.it",
		},
		{
			file:         "anomalia.eml",
			kind:         KindAnomaly,
			cleanSubject: "Hello from outside",
			from:         "someone@ordinary.example.com",
		},
		{
			file:         "ordinary.eml",
			kind:         KindOrdinary,
			cleanSubject: "Weekend plans",
			from:         "friend@example.com",
		},
		{
			file:         "breve-consegna.eml",
			kind:         KindReceipt,
			cleanSubject: "Contratto firmato",
			from:         "alice@pec.example.it",
			refID:        "<client-id@mail.client>",
			receiptTipo:  "avvenuta-consegna",
			consegna:     "bob@pec.altro.it",
			hasAttach:    "contratto.pdf.hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			msg, err := ParseFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if msg.Kind != tt.kind {
				t.Fatalf("kind: got %v want %v", msg.Kind, tt.kind)
			}
			if msg.CleanSubject != tt.cleanSubject {
				t.Fatalf("clean subject: got %q want %q", msg.CleanSubject, tt.cleanSubject)
			}
			if msg.From != tt.from {
				t.Fatalf("from: got %q want %q", msg.From, tt.from)
			}
			if tt.refID != "" && msg.RefMessageID != tt.refID {
				t.Fatalf("ref id: got %q want %q", msg.RefMessageID, tt.refID)
			}
			if msg.HasSignature != tt.hasSig {
				t.Fatalf("has signature: got %v want %v", msg.HasSignature, tt.hasSig)
			}
			if tt.receiptTipo != "" {
				if msg.Receipt == nil {
					t.Fatal("expected Receipt")
				}
				if msg.Receipt.Tipo != tt.receiptTipo {
					t.Fatalf("receipt tipo: got %q want %q", msg.Receipt.Tipo, tt.receiptTipo)
				}
			}
			if tt.errore != "" {
				if msg.Receipt == nil || msg.Receipt.Errore != tt.errore {
					t.Fatalf("errore: got %+v", msg.Receipt)
				}
				if msg.Receipt.ErroreEsteso != "Casella piena" {
					t.Fatalf("errore-esteso: %q", msg.Receipt.ErroreEsteso)
				}
			}
			if tt.consegna != "" {
				if msg.Receipt == nil || msg.Receipt.Consegna != tt.consegna {
					t.Fatalf("consegna: got %+v", msg.Receipt)
				}
			}
			if tt.hasAttach != "" {
				found := false
				for _, a := range msg.Attachments {
					if a.Filename == tt.hasAttach {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("attachment %q not found in %#v", tt.hasAttach, filenames(msg.Attachments))
				}
			}
			// Structural parts must never appear as user attachments.
			for _, a := range msg.Attachments {
				lower := strings.ToLower(a.Filename)
				if lower == "daticert.xml" || lower == "postacert.eml" || strings.HasSuffix(lower, ".p7s") {
					t.Fatalf("structural attachment leaked: %s", a.Filename)
				}
			}
		})
	}
}

func TestTransportBodyFromPostacert(t *testing.T) {
	msg, err := ParseFile("testdata/transport.eml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg.BodyRaw, "contratto firmato") && !strings.Contains(msg.BodyRaw, "in allegato") {
		t.Fatalf("body should come from postacert.eml, got %q", msg.BodyRaw)
	}
	if msg.DatiCert == nil {
		t.Fatal("expected daticert")
	}
	if msg.MessageID != "opec288.20230912095956.1.1@pec.example.it" {
		t.Fatalf("message id: %q", msg.MessageID)
	}
	want := time.Date(2023, 9, 12, 9, 59, 56, 0, time.FixedZone("PEC+0200", 2*3600))
	if !msg.Date.Equal(want) {
		t.Fatalf("date: got %v want %v", msg.Date, want)
	}
}

func TestBreveHashChecksums(t *testing.T) {
	msg, err := ParseFile("testdata/breve-consegna.eml")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Receipt == nil {
		t.Fatal("expected receipt")
	}
	if msg.Receipt.RicevutaTipo != "breve" {
		t.Fatalf("ricevuta tipo: %s", msg.Receipt.RicevutaTipo)
	}
	hash, ok := msg.Receipt.HashChecksums["contratto.pdf"]
	if !ok {
		t.Fatalf("missing hash, got %#v", msg.Receipt.HashChecksums)
	}
	if hash != "da39a3ee5e6b4b0d3255bfef95601890afd80709" {
		t.Fatalf("hash: %s", hash)
	}
}

func TestAnomalyUsesInnerBody(t *testing.T) {
	msg, err := ParseFile("testdata/anomalia.eml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg.BodyRaw, "ordinary email") {
		t.Fatalf("expected inner body, got %q", msg.BodyRaw)
	}
	if msg.Kind.IsService() != true {
		t.Fatal("anomaly should be service")
	}
}

func TestOrdinaryMarkdown(t *testing.T) {
	msg, err := ParseFile("testdata/ordinary.eml")
	if err != nil {
		t.Fatal(err)
	}
	if msg.BodyMarkdown != "Want to go hiking on Saturday?" {
		t.Fatalf("markdown: %q", msg.BodyMarkdown)
	}
}

func filenames(atts []Attachment) []string {
	out := make([]string, len(atts))
	for i, a := range atts {
		out[i] = a.Filename
	}
	return out
}
