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
