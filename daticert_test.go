package pec

import (
	"testing"
	"time"
)

func TestParseDatiCert(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<postacert tipo="avvenuta-consegna" errore="altro">
  <intestazione>
    <mittente>alice@pec.example.it</mittente>
    <destinatari tipo="certificato">bob@pec.altro.it</destinatari>
    <destinatari tipo="esterno">carol@example.com</destinatari>
    <risposte>alice@pec.example.it</risposte>
    <oggetto>Contratto firmato</oggetto>
  </intestazione>
  <dati>
    <gestore-emittente>Altro PEC S.p.A.</gestore-emittente>
    <data zona="+0200"><giorno>12/09/2023</giorno><ora>10:00:10</ora></data>
    <identificativo>opec288.20230912095956.1.1@pec.example.it</identificativo>
    <msgid>&lt;client-id@mail.client&gt;</msgid>
    <ricevuta tipo="sintetica"/>
    <consegna>bob@pec.altro.it</consegna>
    <errore-esteso>Casella piena</errore-esteso>
  </dati>
</postacert>`)

	dc, err := parseDatiCert(xml)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Tipo != "avvenuta-consegna" {
		t.Fatalf("tipo: %s", dc.Tipo)
	}
	if dc.Errore != "altro" {
		t.Fatalf("errore: %s", dc.Errore)
	}
	if dc.Mittente != "alice@pec.example.it" {
		t.Fatalf("mittente: %s", dc.Mittente)
	}
	if len(dc.Destinatari) != 2 {
		t.Fatalf("destinatari: %d", len(dc.Destinatari))
	}
	if dc.Destinatari[1].Tipo != "esterno" || dc.Destinatari[1].Address != "carol@example.com" {
		t.Fatalf("destinatari[1]: %+v", dc.Destinatari[1])
	}
	if dc.MsgID != "<client-id@mail.client>" {
		t.Fatalf("msgid: %q", dc.MsgID)
	}
	if dc.RicevutaTipo != "sintetica" {
		t.Fatalf("ricevuta: %s", dc.RicevutaTipo)
	}
	if dc.Consegna != "bob@pec.altro.it" {
		t.Fatalf("consegna: %s", dc.Consegna)
	}
	if dc.ErroreEsteso != "Casella piena" {
		t.Fatalf("errore-esteso: %s", dc.ErroreEsteso)
	}
	want := time.Date(2023, 9, 12, 10, 0, 10, 0, time.FixedZone("PEC+0200", 2*3600))
	if !dc.Data.Equal(want) {
		t.Fatalf("data: got %v want %v", dc.Data, want)
	}
}
