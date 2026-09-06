VERDICT: BLOCKED

## Sicherheitsbericht

**Hinweis zu Scanner-Befunden:** Es wurden keine Security-Scanner-Ergebnisse geliefert (kein Bandit, pip-audit, npm audit oder Semgrep). Das Fehlen von Scanner-Output ist kein Befund; die folgende Bewertung beruht auf manueller Code-Analyse.

### 1. Fehlende Authentifizierung und Autorisierung für alle API-Endpunkte
- **Schweregrad:** Kritisch
- **Betroffene Stelle:** `main.go` (`newHandler`, Routing) sowie alle Handler in `internal/api/`
- **Beschreibung:** Die REST-API ist vollständig offen. Es gibt weder eine API-Key-/Bearer-Token-Prüfung noch Basic Auth oder mTLS. Jeder Client, der den Server-Port erreicht, kann Feature-Flags anlegen (`POST /flags`), ändern (`PUT /flags/{key}`), löschen (`DELETE /flags/{key}`) und auswerten (`GET /flags/{key}/evaluate`). Ein Angreifer kann das Verhalten der gesteuerten Anwendung manipulieren, z. B. ein Kill-Switch aktivieren oder Rollout-Prozentsätze ändern.
- **Angriffsbeispiel:**
  ```
  curl -X POST http://<host>:8080/flags \
       -H 'Content-Type: application/json' \
       -d '{"key":"critical","enabled":true,"rollout_percent":100}'
  ```
  Oder:
  ```
  curl -X DELETE http://<host>:8080/flags/critical
  ```
- **Fix:**
  - Eine Authentifizierungs-Middleware vor die API schalten, die z. B. einen konfigurierbaren Bearer-Token prüft:
    ```go
    func Auth(requiredToken string, next http.Handler) http.Handler { ... }
    ```
  - Den Token aus `os.Getenv("FLAG_API_TOKEN")` beziehen und im Fehlerfall `401` mit JSON-Fehlerobjekt zurückgeben.
  - Für Schreib-/Lese-Trennung optional eine rollenbasierte Berechtigung (z. B. nur Lesezugriff ohne Token für `GET /healthz`, voller Zugriff nur mit gültigem Token).
  - Tests entsprechend erweitern (`main_test.go`, Handler-Tests).

### 2. Transportverschlüsselung fehlt (HTTP statt HTTPS)
- **Schweregrad:** Mittel
- **Betroffene Stelle:** `main.go` (`newServer`, `ListenAndServe`)
- **Beschreibung:** Der Server bindet standardmäßig an `:8080` und verwendet keine TLS-Verschlüsselung. Die `user`-ID wird bei `GET /flags/{key}/evaluate?user=…` als Query-Parameter übertragen. Ohne TLS können Nutzer-IDs und Flag-Beschreibungen im Klartext abgefangen werden, sofern der Dienst nicht durch einen TLS-terminierenden Reverse-Proxy geschützt wird.
- **Fix:**
  - Entweder TLS direkt im Server aktivieren (`ListenAndServeTLS`) und Zertifikate konfigurieren, oder
  - im Deployment ausdrücklich vorsehen, dass der Service ausschließlich hinter einem TLS-Reverse-Proxy läuft (z. B. `ADDR` auf `127.0.0.1:8080` binden und Proxy auf `localhost`).
  - Empfehlung: Standardbindung auf Loopback (`127.0.0.1:8080`), um versehentliche öffentliche Exposition zu vermeiden.

### 3. Pfadparameter `{key}` wird außerhalb des POST-/PUT-Pfads nicht auf gültige Zeichen geprüft
- **Schweregrad:** Niedrig
- **Betroffene Stellen:** `internal/api/evaluate.go`, `internal/api/flags.go` (`GetFlag`, `UpdateFlag`, `DeleteFlag`)
- **Beschreibung:** `validKey` wird nur beim Anlegen (`CreateFlag`) angewendet. Die Schlüssel aus `r.PathValue("key")` in `GET`, `PUT`, `DELETE` und `evaluate` werden ungeprüft an den Store übergeben. Da im Store durch den Create-Pfad keine ungültigen Keys entstehen können, ist das aktuell nicht direkt ausnutzbar. Allerdings ist es inkonsistent und kann bei späteren Erweiterungen oder durch Pfadnormalisierungen (`/flags/..`) zu Problemen führen.
- **Fix:**
  - `validKey` auch in diesen Handlern anwenden und bei ungültigem Key `400` mit JSON-Fehlerobjekt zurückgeben, statt den Store zu konsultieren.

### 4. Hash-Kollision durch ungeschützte String-Konkatenation in der Rollout-Entscheidung
- **Schweregrad:** Niedrig
- **Betroffene Stellen:** `internal/evaluate/evaluate.go` (`Decide`), `internal/api/evaluate.go` (`evaluateFlag`)
- **Beschreibung:** Die Hash-Eingabe wird als `key + ":" + user` gebildet. Da `user` beliebige Zeichen enthalten darf, können verschiedene Paare dieselbe Eingabe erzeugen, z. B. `key="a", user="b:c"` und `key="a:b", user="c"`. Das führt zu identischen Rollout-Entscheidungen für unterschiedliche logische Paare. Es handelt sich nicht um einen direkten Sicherheitsangriff, aber um eine Schwäche in der Verteilungslogik.
- **Fix:**
  - Eine eindeutige Serialisierung verwenden, z. B. Längenpräfixe:
    ```go
    fmt.Sprintf("%d:%s:%d:%s", len(key), key, len(user), user)
    ```
  - Alternativ `encoding/json` auf eine kleine Struct anwenden.

### 5. Unbegrenztes Speicherwachstum des In-Memory-Stores
- **Schweregrad:** Niedrig
- **Betroffene Stelle:** `internal/store/store.go`
- **Beschreibung:** Der Store hält alle Flags unbegrenzt im Speicher; es gibt keine Maximalzahl oder Quota. In Verbindung mit der fehlenden Authentifizierung (Finding 1) kann ein Angreifer beliebig viele Flags anlegen und so Speicher und CPU belasten. Nach Einführung der Auth sollte zusätzlich ein Limit oder ein Rate-Limit erwogen werden.
- **Fix:**
  - Maximalanzahl von Flags konfigurierbar machen (z. B. 1000) und bei Überschreitung `409` oder `429` zurückgeben.
  - Optional ein Request-Rate-Limit in der Middleware.

## Weitere Beobachtungen
- **Dependencies:** `go.mod` enthält laut Projektdatei nur Modul- und Go-Version; keine externen Abhängigkeiten und damit keine bekannten Schwachstellen durch Pakete. Kein Handlungsbedarf.
- **RequestBody-Limit:** `http.MaxBytesReader` in `decodeJSON` begrenzt POST/PUT korrekt auf 1 MiB und liefert `413`. Das ist sauber implementiert.
- **Logging:** Die Middleware protokolliert ausschließlich Methode, Pfad ohne Query und Statuscode. Es werden keine `user`-IDs oder Request-Bodys geloggt. Das entspricht AC-16/AC-17.
- **Server-Timeout-Konfiguration:** `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout` sind gesetzt und mindern Slowloris-Angriffe.

## Fazit
Der Code ist in sich konsistent und erfüllt viele geforderte Sicherheitsaspekte (Timeout, Body-Limit, Key-Validierung beim Create, datenschutzfreundliches Logging). Der **kritische Mangel ist die vollständig fehlende Authentifizierung/Autorisierung**, die es jedem Netzwerkteilnehmer erlaubt, Feature-Flags zu manipulieren. Dies ist für ein Produkt, das Anwendungsverhalten steuert, nicht akzeptabel. Zudem sollte die Transportverschlüsselung sichergestellt werden. Daher wird das Produkt aktuell **blockiert**.