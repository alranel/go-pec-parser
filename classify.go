package pec

import "strings"

// classify determines the message Kind from headers, daticert, and subject.
// Order: X-Trasporto → X-Ricevuta → postacert@tipo → subject prefix → ordinary.
func classify(h PECHeaders, dc *DatiCert, subject string) Kind {
	trasporto := strings.ToLower(strings.TrimSpace(h.Trasporto))
	switch trasporto {
	case "posta-certificata":
		return KindTransport
	case "errore":
		return KindAnomaly
	}

	ricevuta := strings.ToLower(strings.TrimSpace(h.Ricevuta))
	if isReceiptType(ricevuta) {
		return KindReceipt
	}

	if dc != nil {
		tipo := strings.ToLower(strings.TrimSpace(dc.Tipo))
		switch tipo {
		case "posta-certificata":
			return KindTransport
		case "accettazione", "non-accettazione", "presa-in-carico",
			"avvenuta-consegna", "errore-consegna", "preavviso-errore-consegna",
			"rilevazione-virus":
			return KindReceipt
		}
	}

	return subjectImpliesKind(subject)
}

func isReceiptType(v string) bool {
	switch v {
	case "accettazione", "non-accettazione", "presa-in-carico",
		"avvenuta-consegna", "errore-consegna", "preavviso-errore-consegna",
		"rilevazione-virus":
		return true
	default:
		return false
	}
}

// receiptTipo returns the receipt/event type string for service messages.
func receiptTipo(h PECHeaders, dc *DatiCert, kind Kind) string {
	if dc != nil && dc.Tipo != "" {
		return dc.Tipo
	}
	if h.Ricevuta != "" {
		return h.Ricevuta
	}
	if kind == KindAnomaly {
		return "anomalia"
	}
	if kind == KindTransport {
		return "posta-certificata"
	}
	return ""
}
