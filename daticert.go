package pec

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type xmlPostacert struct {
	XMLName      xml.Name        `xml:"postacert"`
	Tipo         string          `xml:"tipo,attr"`
	Errore       string          `xml:"errore,attr"`
	Intestazione xmlIntestazione `xml:"intestazione"`
	Dati         xmlDati         `xml:"dati"`
}

type xmlIntestazione struct {
	Mittente    string           `xml:"mittente"`
	Destinatari []xmlDestinatari `xml:"destinatari"`
	Risposte    string           `xml:"risposte"`
	Oggetto     string           `xml:"oggetto"`
}

type xmlDestinatari struct {
	Tipo    string `xml:"tipo,attr"`
	Address string `xml:",chardata"`
}

type xmlDati struct {
	GestoreEmittente string       `xml:"gestore-emittente"`
	Data             xmlData      `xml:"data"`
	Identificativo   string       `xml:"identificativo"`
	MsgID            string       `xml:"msgid"`
	Ricevuta         *xmlRicevuta `xml:"ricevuta"`
	Consegna         string       `xml:"consegna"`
	Ricezione        []string     `xml:"ricezione"`
	ErroreEsteso     string       `xml:"errore-esteso"`
}

type xmlData struct {
	Zona   string `xml:"zona,attr"`
	Giorno string `xml:"giorno"`
	Ora    string `xml:"ora"`
}

type xmlRicevuta struct {
	Tipo string `xml:"tipo,attr"`
}

// parseDatiCert parses daticert.xml bytes into DatiCert.
func parseDatiCert(data []byte) (*DatiCert, error) {
	var raw xmlPostacert
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("daticert.xml: %w", err)
	}

	dc := &DatiCert{
		Tipo:             strings.TrimSpace(raw.Tipo),
		Errore:           strings.TrimSpace(raw.Errore),
		Mittente:         strings.TrimSpace(raw.Intestazione.Mittente),
		Risposte:         strings.TrimSpace(raw.Intestazione.Risposte),
		Oggetto:          strings.TrimSpace(raw.Intestazione.Oggetto),
		GestoreEmittente: strings.TrimSpace(raw.Dati.GestoreEmittente),
		Identificativo:   strings.TrimSpace(raw.Dati.Identificativo),
		MsgID:            strings.TrimSpace(raw.Dati.MsgID),
		Consegna:         strings.TrimSpace(raw.Dati.Consegna),
		ErroreEsteso:     strings.TrimSpace(raw.Dati.ErroreEsteso),
	}
	if dc.Errore == "" {
		dc.Errore = "nessuno"
	}
	if raw.Dati.Ricevuta != nil {
		dc.RicevutaTipo = strings.TrimSpace(raw.Dati.Ricevuta.Tipo)
	}
	for _, d := range raw.Intestazione.Destinatari {
		tipo := strings.TrimSpace(d.Tipo)
		if tipo == "" {
			tipo = "certificato"
		}
		dc.Destinatari = append(dc.Destinatari, Destinatario{
			Address: strings.TrimSpace(d.Address),
			Tipo:    tipo,
		})
	}
	for _, r := range raw.Dati.Ricezione {
		if s := strings.TrimSpace(r); s != "" {
			dc.Ricezione = append(dc.Ricezione, s)
		}
	}

	t, err := parseDatiCertDate(raw.Dati.Data)
	if err != nil {
		return nil, err
	}
	dc.Data = t
	return dc, nil
}

func parseDatiCertDate(d xmlData) (time.Time, error) {
	giorno := strings.TrimSpace(d.Giorno)
	ora := strings.TrimSpace(d.Ora)
	zona := strings.TrimSpace(d.Zona)
	if giorno == "" || ora == "" {
		return time.Time{}, nil
	}
	if zona == "" {
		zona = "+0000"
	}
	// zona is [+|-]hhmm; Go expects [+|-]hhmm for -0700 layout with RFC822Z-like forms.
	loc, err := parseZona(zona)
	if err != nil {
		return time.Time{}, fmt.Errorf("daticert data zona %q: %w", zona, err)
	}
	t, err := time.ParseInLocation("02/01/2006 15:04:05", giorno+" "+ora, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("daticert data: %w", err)
	}
	return t, nil
}

func parseZona(zona string) (*time.Location, error) {
	if len(zona) != 5 || (zona[0] != '+' && zona[0] != '-') {
		return nil, fmt.Errorf("invalid zona format")
	}
	sign := 1
	if zona[0] == '-' {
		sign = -1
	}
	hh, err := strconv.Atoi(zona[1:3])
	if err != nil {
		return nil, err
	}
	mm, err := strconv.Atoi(zona[3:5])
	if err != nil {
		return nil, err
	}
	offset := sign * (hh*3600 + mm*60)
	return time.FixedZone("PEC"+zona, offset), nil
}
