# Werkstatt-Kundenportal

Ein Kundenportal für eine Kfz-Werkstatt. Kunden fragen über die Web-App einen
Termin an, verfolgen den Status ihres Auftrags und rufen die fertige Rechnung ab.
Mitarbeiter melden sich an, pflegen Aufträge und sehen ein Dashboard mit offenen
Aufträgen, heute fertiggestellten Aufträgen und dem Monatsumsatz.

## Architektur

Das Produkt besteht aus drei Diensten:

- **api** (`backend/`, Go): HTTP-API auf Basis von `net/http` aus der
  Standardbibliothek, Zugriff auf PostgreSQL über `pgx`.
- **worker** (`worker/`, Python): liest Auftragsnummern aus einer
  Valkey-Warteschlange und erzeugt die Rechnungen.
- **web** (`frontend/`, Vite + React + TypeScript): spricht ausschließlich mit
  der API.

## Tech-Stack

- Go (aktuell), `net/http`, `pgx` (`github.com/jackc/pgx/v5`)
- PostgreSQL 18 (echte Datenbank, kein SQLite)
- Valkey 9.1 als Warteschlange und Cache
- Python (Worker), Vite + React + TypeScript (Web)
- Tests: Go `testing` + `httptest` gegen PostgreSQL, `pytest`, Vitest

## Datenbanken lokal starten

Ein frischer Clone braucht PostgreSQL und Valkey. Ein Befehl startet beide:

```sh
docker compose up -d
```

Das startet `postgres:18.6-trixie` auf Port `5432` (Benutzer und Datenbank
`app`, lokale Vertrauens-Authentifizierung wie in der CI) und
`valkey/valkey:9.1.2-trixie` auf Port `6379`.

## Installation und Start

```sh
cd backend
go mod download
```

Die API liest ihre Konfiguration aus der Umgebung. Vor dem Start exportieren:

```sh
export DATABASE_URL="postgresql://app@localhost:5432/app"
export VALKEY_URL="redis://localhost:6379/0"
export PORT=8080
export HOURLY_RATE_CENTS=9500
export CORS_ORIGIN="http://localhost:5173"
export INITIAL_EMPLOYEE_EMAIL="chef@beispiel.de"
export INITIAL_EMPLOYEE_PASSWORD="ein-sicheres-passwort"

go run ./cmd/api
```

Beim Start führt die API automatisch alle Migrationen aus
(`backend/migrations/0001_init.sql`) und legt – sofern noch kein Mitarbeiter
existiert und die beiden `INITIAL_EMPLOYEE_*`-Variablen gesetzt sind – den ersten
Mitarbeiter an.

Für die Produktion bauen:

```sh
cd backend
go build -o bin/api ./cmd/api
./bin/api
```

## Umgebungsvariablen

### api

| Variable | Beschreibung | Pflicht |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL-Verbindungsadresse | ja |
| `VALKEY_URL` | Valkey-Adresse (Warteschlange) | ja |
| `PORT` | HTTP-Port (Standard `8080`) | nein |
| `HOURLY_RATE_CENTS` | Stundensatz in Cent (Standard `9500`) | nein |
| `INVOICE_QUEUE_KEY` | Name der Valkey-Liste (Standard `invoices`) | nein |
| `CORS_ORIGIN` | erlaubter Origin der Web-App | nein |
| `INITIAL_EMPLOYEE_EMAIL` | E-Mail des ersten Mitarbeiters | nein |
| `INITIAL_EMPLOYEE_PASSWORD` | Passwort des ersten Mitarbeiters | nein |

`DATABASE_URL`, `VALKEY_URL` und die Zugangsdaten des ersten Mitarbeiters sind
Geheimnisse: sie haben keinen literalen Wert im Repository, sondern werden in
`RUN.json` als Laufzeitwerte deklariert.

### worker

| Variable | Beschreibung |
| --- | --- |
| `DATABASE_URL` | PostgreSQL-Verbindungsadresse |
| `VALKEY_URL` | Valkey-Adresse |
| `HOURLY_RATE_CENTS` | Stundensatz in Cent |
| `VAT_PERCENT` | Mehrwertsteuersatz in Prozent (19) |
| `INVOICE_QUEUE_KEY` | Name der Valkey-Liste |

### web

| Variable | Beschreibung |
| --- | --- |
| `VITE_API_BASE_URL` | Basis-URL der API |

## API-Endpunkte

Alle Antworten sind JSON. Jeder Fehler hat die Form
`{"error":{"code":"...","message":"..."}}` mit den Codes `validation_error`,
`not_found`, `conflict`, `invalid_transition`, `unauthorized`, `rate_limited`
und `internal_error`.

### Öffentlich

| Methode | Pfad | Body | Antwort |
| --- | --- | --- | --- |
| GET | `/api/health` | – | `200 {"status":"ok"}` |
| POST | `/api/customers` | `{"name","email","phone"}` | `201 {"id","name","email","phone"}` |
| POST | `/api/vehicles` | `{"license_plate","make","model","mileage_km"}` | `201 {...}`; `409` bei doppeltem Kennzeichen |
| POST | `/api/orders` | `{"customer":{...},"vehicle":{...},"preferred_date","problem_description"}` | `201 {"order_number","status","created_at"}` |
| GET | `/api/orders/status?order_number=&license_plate=` | – | `200 {"order_number","status","status_history":[...]}`; `404` |
| GET | `/api/orders/{order_number}/invoice?license_plate=` | – | `200 {"order_number","items":[...],"net_cents","vat_cents","gross_cents","created_at"}`; `404` |
| POST | `/api/auth/login` | `{"email","password"}` | `200 {"token","employee":{...}}`; `401`; `429` |

### Werkstatt (erfordert `Authorization: Bearer <token>`)

| Methode | Pfad | Antwort |
| --- | --- | --- |
| GET | `/api/workshop/orders?status=&license_plate=` | `200 {"orders":[...]}` |
| GET | `/api/workshop/orders/{order_number}` | `200 OrderDetail`; `404` |
| POST | `/api/workshop/orders/{order_number}/confirm` | `200 OrderDetail`; `409` |
| POST | `/api/workshop/orders/{order_number}/items` | `200 OrderDetail`; `400` |
| POST | `/api/workshop/orders/{order_number}/status` | `200 OrderDetail`; `400 invalid_transition` |
| GET | `/api/workshop/dashboard` | `200 {"open_orders","completed_today","revenue_month_cents"}` |

`Status` ist einer von `angefragt`, `bestaetigt`, `in_arbeit`, `fertig`,
`abgeholt`. Erlaubt ist nur der Übergang entlang dieser Reihenfolge.

### Web-Oberfläche

| Pfad | Zweck |
| --- | --- |
| `/` | Startseite für Kunden |
| `/status` | Auftragsstatus abfragen |
| `/rechnung` | Rechnung abrufen |
| `/werkstatt/login` | Mitarbeiter-Anmeldung |
| `/werkstatt/auftraege` | Auftragsliste |
| `/werkstatt/auftraege/:orderNumber` | Auftragsdetails |
| `/werkstatt/dashboard` | Dashboard |
| `/impressum`, `/datenschutz` | Rechtliches |

## Tests

```sh
cd backend
go test ./...
```

Die Tests laufen mit `httptest` gegen eine echte PostgreSQL-Datenbank; es wird
kein SQLite verwendet. `DATABASE_URL` muss dafür gesetzt sein.

## Warteschlange

Wird ein Auftrag auf `fertig` gesetzt, legt die API `{"order_number":"..."}`
per `LPUSH` in die Valkey-Liste `INVOICE_QUEUE_KEY` (Standard `invoices`). Der
Worker entnimmt die Nachricht und erzeugt die Rechnung.

## Funktionsumfang

- Kundenanlage und Fahrzeuganlage mit Dublettenprüfung beim Kennzeichen
- Auftragsanfrage mit Wunschtermin und Problembeschreibung
- Statusverfolgung mit lückenlosem Statusprotokoll
- Rechnungsabruf inklusive Positionen, Netto, Mehrwertsteuer (19 %) und Brutto
- Mitarbeiter-Anmeldung mit gehashten Passwörtern und Rate-Limit
- Auftragsliste mit Statusfilter und Kennzeichensuche
- Positionen aus Arbeitszeit und Teilen erfassen
- Dashboard mit offenen Aufträgen, heutigen Abschlüssen und Monatsumsatz
- asynchroner Rechnungslauf über Valkey
- Impressum und Datenschutzerklärung
