package pec

import "testing"

func TestCleanSubject(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"POSTA CERTIFICATA: Contratto firmato", "Contratto firmato"},
		{"ACCETTAZIONE: Hello", "Hello"},
		{"CONSEGNA: Hello", "Hello"},
		{"ANOMALIA MESSAGGIO: Hello", "Hello"},
		{"AVVISO DI MANCATA CONSEGNA PER VIRUS: X", "X"},
		{"AVVISO DI MANCATA CONSEGNA PER SUP. TEMPO MASSIMO: X", "X"},
		{"posta certificata: mixed case", "mixed case"},
		{"No prefix here", "No prefix here"},
		{"  ACCETTAZIONE:  spaced  ", "spaced"},
	}
	for _, tt := range tests {
		got := CleanSubject(tt.in)
		if got != tt.want {
			t.Errorf("CleanSubject(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSubjectImpliesKind(t *testing.T) {
	if subjectImpliesKind("POSTA CERTIFICATA: x") != KindTransport {
		t.Fatal("expected transport")
	}
	if subjectImpliesKind("ACCETTAZIONE: x") != KindReceipt {
		t.Fatal("expected receipt")
	}
	if subjectImpliesKind("ANOMALIA MESSAGGIO: x") != KindAnomaly {
		t.Fatal("expected anomaly")
	}
	if subjectImpliesKind("Hello") != KindOrdinary {
		t.Fatal("expected ordinary")
	}
}
