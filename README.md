# Feature-Flag-Service REST-API

Ein Feature-Flag-Service als REST-API in Go, ausschließlich mit der
Standardbibliothek `net/http`. Flags werden thread-sicher in einem
In-Memory-Store gehalten und über CRUD-Endpunkte verwaltet; ein
Evaluate-Endpunkt liefert anhand eines stabilen Hashs aus Flag-Key und
Nutzer-ID eine deterministische Ja/Nein-Entscheidung gemäß `rollout_percent`.

## Tech Stack

- Sprache: Go (≥ 1.22)
- Framework: `net/http` (Standardbibliothek)
- Tests: `net/http/httptest`
- Datenspeicher: In-Memory (thread-sicher über `sync.RWMutex`)

## Installation

Voraussetzung: Go 1.22 oder neuer. Es werden keine externen Module benötigt.

```bash
go build ./...
```

## Ausführen (Entwicklung)

```bash
go run .
```

Der Server startet auf `ADDR` (Standard `:8080`). Beispiel mit anderer Adresse:

```bash
ADDR=:9090 go run .
```

## Testen

```bash
go test ./...
```

## Endpunkt-Übersicht

| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | `/healthz` | Health-Check, antwortet mit `200` und `{"status":"ok"}` |
| POST | `/flags` | Legt ein Flag an |
| GET | `/flags` | Listet alle Flags |
| GET | `/flags/{key}` | Liefert ein einzelnes Flag |
| PUT | `/flags/{key}` | Aktualisiert ein Flag |
| DELETE | `/flags/{key}` | Entfernt ein Flag |
| GET | `/flags/{key}/evaluate?user={id}` | Deterministische Rollout-Entscheidung |

Alle Fehlerantworten sind JSON-Objekte der Form `{"error":"..."}`. Unbekannte
Pfade antworten mit `404`, falsche Methoden mit `405` (jeweils als
JSON-Fehlerobjekt).

## Feature-Liste

- Lauffähiges HTTP-Gerüst mit Health-Endpoint
- Methodenbasierte Routenregistrierung (`GET`, `POST`, `PUT`, `DELETE`)
- In-Memory-Flag-Store mit Thread-Sicherheit
- Deterministischer Rollout über einen FNV-1a-Hash
- Einheitliche JSON-Fehlerobjekte
- Logging-Middleware
- Go-Tests mit `httptest`

## Konfiguration

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `ADDR` | `:8080` | Adresse, auf der der HTTP-Server lauscht |

## Datenschutz & Rechtsgrundlage

- `user` wird ausschließlich transient zur deterministischen
  Rollout-Entscheidung (`GET /flags/{key}/evaluate`) verarbeitet. Die Kennung
  wird nicht persistiert, nicht geloggt und nicht an Dritte weitergegeben.
- Wird der Dienst als Auftragsverarbeiter für einen aufrufenden Dienst
  betrieben, ist ein Auftragsverarbeitungsvertrag (AVV) erforderlich.
- Der aufrufende Dienst trägt die datenschutzrechtliche Verantwortung
  gegenüber der betroffenen Person; dieses Backend stellt keine
  Benutzeroberfläche und keine Datenschutzerklärung bereit.
- Es werden keine Nutzer-IDs und keine personenbezogenen Daten dauerhaft
  gespeichert; Betroffenenrechte (Auskunft, Berichtigung, Löschung) sind
  daher auf Anwendungsebene des aufrufenden Dienstes umzusetzen.

## Security

- **Server-Timeouts:** `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` und
  `IdleTimeout` sind gesetzt und mindern u. a. Slowloris-Angriffe.
- **Body-Limit:** Schreibende Endpunkte (`POST`, `PUT`) begrenzen den
  Request-Body auf 1 MiB und antworten bei Überschreitung mit `413`.
- **Key-Validierung:** Flag-Keys werden validiert; ungültige Keys werden mit
  `400` als JSON-Fehlerobjekt abgewiesen.
- **Logging:** Es werden ausschließlich Methode, Pfad (ohne Query-String) und
  Statuscode protokolliert — keine Query-Parameter, keine Nutzer-IDs und
  keine Request-Bodys.
- **Authentifizierung:** Die API-Endpunkte sind über ein Bearer-Token
  geschützt, das aus der Umgebungsvariable `FLAG_API_TOKEN` bezogen wird.
  Anfragen ohne gültiges Token werden mit `401` als JSON-Fehlerobjekt
  beantwortet.
- **Transport:** Der Betrieb erfolgt ausschließlich hinter einem
  TLS-terminierenden Reverse-Proxy. Der Dienst stellt selbst kein TLS zur
  Verfügung; die Verschlüsselung übernimmt der vorgelagerte Proxy.
- **Standardbindung:** `127.0.0.1:8080` (nur Loopback), um eine versehentliche
  öffentliche Exposition zu vermeiden. Die Adresse ist über `ADDR`
  konfigurierbar.
- **SBOM:** `go.mod`/`go.sum` dienen als Abhängigkeitsnachweis (Software Bill
  of Materials). Aktuell werden keine externen Module verwendet; `go.mod`
  enthält nur die Modul- und Go-Version ohne Fremdabhängigkeiten.
- **Update-/Patch-Prozess:** Der Dienst verwendet aktuell keine externen
  Module, daher bestehen keine bekannten Schwachstellen aus Fremdpaketen.
  Wird künftig eine Abhängigkeit hinzugefügt, wird sie in `go.mod`/`go.sum`
  fixiert und bei Bekanntwerden einer Schwachstelle zeitnah durch eine
  gepatchte Version ersetzt. Vor jeder Veröffentlichung laufen
  `go build ./...` und `go test ./...` in der CI-Pipeline;
  sicherheitsrelevante Korrekturen werden als neue Version ausgeliefert und
  sind beim Betrieb zeitnah einzuspielen.
