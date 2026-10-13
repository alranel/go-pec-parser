package pec

import (
	"strings"
)

// Known PEC subject prefixes, longest first so stripping is unambiguous.
var subjectPrefixes = []string{
	"AVVISO DI MANCATA CONSEGNA PER SUP. TEMPO MASSIMO:",
	"AVVISO DI MANCATA CONSEGNA PER VIRUS:",
	"AVVISO DI NON ACCETTAZIONE PER VIRUS:",
	"AVVISO DI MANCATA CONSEGNA:",
	"AVVISO DI NON ACCETTAZIONE:",
	"ANOMALIA MESSAGGIO:",
	"PROBLEMA DI SICUREZZA:",
	"POSTA CERTIFICATA:",
	"PRESA IN CARICO:",
	"ACCETTAZIONE:",
	"CONSEGNA:",
}

// CleanSubject strips a known PEC subject prefix from s.
// If no prefix matches, s is returned trimmed.
func CleanSubject(s string) string {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	for _, prefix := range subjectPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return strings.TrimSpace(s[len(prefix):])
		}
	}
	return s
}

// subjectImpliesKind returns a Kind inferred from a PEC subject prefix, or KindOrdinary.
func subjectImpliesKind(subject string) Kind {
	upper := strings.ToUpper(strings.TrimSpace(subject))
	switch {
	case strings.HasPrefix(upper, "POSTA CERTIFICATA:"):
		return KindTransport
	case strings.HasPrefix(upper, "ANOMALIA MESSAGGIO:"):
		return KindAnomaly
	case strings.HasPrefix(upper, "ACCETTAZIONE:"),
		strings.HasPrefix(upper, "CONSEGNA:"),
		strings.HasPrefix(upper, "PRESA IN CARICO:"),
		strings.HasPrefix(upper, "PROBLEMA DI SICUREZZA:"),
		strings.HasPrefix(upper, "AVVISO DI NON ACCETTAZIONE"),
		strings.HasPrefix(upper, "AVVISO DI MANCATA CONSEGNA"):
		return KindReceipt
	default:
		return KindOrdinary
	}
}
