# go-pec-parser

Go package that parses Italian **PEC** (*Posta Elettronica Certificata*) e-mail messages.

Given a raw RFC 822 / MIME message, it extracts:

- Clean subject (PEC prefixes stripped; prefers `daticert.xml` `<oggetto>`)
- Sender, date/time, Message-ID
- Kind: transport envelope, receipt/notice, anomaly, or ordinary mail
- Body (raw text and Markdown)
- User attachments (structural PEC parts excluded)
- For service messages: referenced original Message-ID and certification metadata (`daticert.xml`)

Machine-readable certification data is taken from **`daticert.xml`** (AgID Regole Tecniche / [RFC 6109](https://datatracker.ietf.org/doc/rfc6109/)), not from the human-readable body.

## Install

```bash
go get github.com/alranel/go-pec-parser
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	pec "github.com/alranel/go-pec-parser"
)

func main() {
	msg, err := pec.ParseFile("message.eml")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(msg.Kind, msg.CleanSubject)
	fmt.Println("From:", msg.From)
	fmt.Println("Message-ID:", msg.MessageID)
	if msg.Kind.IsService() {
		fmt.Println("Ref Message-ID:", msg.RefMessageID)
		if msg.Receipt != nil {
			fmt.Println("Event:", msg.Receipt.Tipo, msg.Receipt.Errore)
		}
	}
	fmt.Println(msg.BodyMarkdown)
}
```

## Classification

| Kind | Meaning |
|------|---------|
| `KindTransport` | `X-Trasporto: posta-certificata` — certified content |
| `KindReceipt` | Accettazione, consegna, errori, virus, presa-in-carico, … |
| `KindAnomaly` | `X-Trasporto: errore` — anomaly wrapper |
| `KindOrdinary` | Not a PEC system message |

Order: `X-Trasporto` → `X-Ricevuta` → `postacert@tipo` → subject prefix → ordinary.

## Legal / certification fields

When `daticert.xml` is present, `Message.DatiCert` (and `Message.Receipt` for service messages) exposes:

| Field | Role |
|-------|------|
| `Identificativo` | PEC system message id |
| `MsgID` | Original client Message-ID (before Access Point substitution) |
| `GestoreEmittente` | Issuing PEC provider |
| `Data` | Certified event timestamp |
| `Errore` / `ErroreEsteso` | Error class and detail |
| `Consegna` | Recipient address on delivery / non-delivery receipts |
| `HashChecksums` | SHA-1 hex for attachments in *breve* delivery receipts |

S/MIME signature **verification** (Indice dei gestori) is out of scope; `HasSignature` only reports that a signature part was present.

## License

[MIT](LICENSE)
