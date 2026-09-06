VERDICT: CHANGES_REQUESTED

## Gesamtbewertung

Der vorgelegte Stand ist ein solides, datenschutzbewusstes Go-Backend: Es gibt keine Persistenz personenbezogener Daten, das Logging verzichtet auf Query-Strings und Nutzer-IDs, der Server setzt Timeouts, Body-Limits und eine Fail-closed-Authentifizierung. Es bestehen jedoch behebbare Lücken bei Transportverschlüsselung, Datenschutzdokumentation, CRA-Nachweisen (SBOM/Update-Prozess), einem nicht konstantzeitigen Token-Vergleich und einer Abweichung von AC-08. Kein fundamentaler, nicht behebbarer Verstoß — daher `CHANGES_REQUESTED`.

---

## 1. GDPR / Datenschutz

### Befund 1.1 — Fehlende Transportverschlüsselung (TLS)
- **Schwere:** hoch
- **Fundstelle:** `main.go`, `newServer()` / `main()`: Es wird ausschließlich `ListenAndServe()` ohne TLS-Konfiguration verwendet. Der Dienst verarbeitet `X-User-ID` (personenbezogenes Datum) und einen `FLAG_API_TOKEN` (Zugangsgeheimnis). Wird `ADDR` auf eine öffentliche Schnittstelle gesetzt, erfolgt die Übertragung im Klartext.
- **Begründung:** Art. 32 DSGVO verlangt geeignete technische und organisatorische Maßnahmen, einschließlich Verschlüsselung bei Übertragung personenbezogener Daten. Bearer-Token und Nutzer-IDs sind schützenswert.
- **Maßnahme (konkret):**
  - In `main.go` Umgebungsvariablen `TLS_CERT_FILE` und `TLS_KEY_FILE` einführen.
  - Wenn beide gesetzt sind: `srv.ListenAndServeTLS(tlsCertFile, tlsKeyFile)` verwenden.
  - Zusätzlich in `README.md` / `SECURITY.md` verbindlich dokumentieren: „Der Dienst darf ohne TLS nur auf `127.0.0.1` betrieben werden. Bei Betrieb hinter einem TLS-terminierenden Reverse Proxy muss die Weiterleitung ausschließlich über HTTPS erfolgen.“
  - Optional: `TLSConfig` mit mindestens TLS 1.2 und sicheren Cipher-Suiten setzen.

### Befund 1.2 — Rechtsgrundlage und Datenschutzdokumentation nicht sichtbar
- **Schwere:** mittel
- **Fundstelle:** `internal/api/evaluate.go` verarbeitet `X-User-ID`; `README.md`, `COMPLIANCE.md` und `SECURITY.md` sind vorhanden, ihre Inhalte sind aber nicht Teil des sichtbaren Standes. Eine Datenschutz-/Verarbeitungsdokumentation ist im Code nicht erkennbar.
- **Begründung:** Die Verarbeitung der Nutzer-ID zur Feature-Auswertung benötigt eine dokumentierte Rechtsgrundlage (je nach Einsatz Art. 6 Abs. 1 lit. b DSGVO bei Vertragserfüllung oder lit. f DSGVO bei berechtigtem Interesse). Betreiber und ggf. Auftragsverarbeiter müssen die Verarbeitung nachweisen können.
- **Maßnahme (konkret):**
  - In `README.md` oder `COMPLIANCE.md` einen Abschnitt „Datenschutz“ ergänzen mit:
    - Verarbeitete Daten: `X-User-ID` (transient), `Authorization`-Header (nur für Authentifizierung), keine IP-Adresse in Access-Logs.
    - Zweck: deterministische Feature-Flag-Auswertung.
    - Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung) bzw. lit. f DSGVO (berechtigtes Interesse), je nach Bereitstellungsmodell.
    - Speicherdauer: keine dauerhafte Speicherung; nur flüchtige In-Memory-Berechnung.
    - Betroffenenrechte: Da keine Speicherung erfolgt, entfallen Lösch-/Auskunftspflichten; Hinweis auf vorgelagertes TLS.
    - Auftragsverarbeitung: Wenn der Service als Auftragsverarbeiter betrieben wird, muss ein AV-Vertrag geschlossen werden.

### Befund 1.3 — API-Vertrag für `user` widerspricht AC-08 (und hat Datenschutzbezug)
- **Schwere:** hoch (Marktreife/AC-Konformität)
- **Fundstelle:** `internal/api/evaluate.go` liest `user` ausschließlich aus dem Header `X-User-ID`. Die Sprint-Spec AC-08 verlangt jedoch `GET /flags/{key}/evaluate?user={id}` per Query-Parameter.
- **Begründung:** Ein Client, der die Spec befolgt, erhält `400 {"error":"user is required"}`. Das Produkt erfüllt damit ein zentrales Abnahmekriterium nicht. Datenschutzrechtlich ist der Header die besser geschützte Variante, aber die API muss den vereinbarten Vertrag erfüllen.
- **Maßnahme (konkret):**
  - In `internal/api/evaluate.go` zuerst `r.URL.Query().Get("user")` auswerten, bei leerem Wert auf `r.Header.Get("X-User-ID")` zurückfallen:
    ```go
    user := r.URL.Query().Get("user")
    if user == "" {
        user = r.Header.Get("X-User-ID")
    }
    ```
  - In `README.md` dokumentieren, dass aus Datenschutzgründen der Header `X-User-ID` bevorzugt wird, Query-Parameter aber aus Kompatibilität unterstützt werden.
  - Tests in `internal/api/evaluate_test.go` und `main_test.go` um einen Query-Parameter-Fall ergänzen.

### Positiv (GDPR)
- Logging in `internal/middleware/logging.go` protokolliert ausschließlich Methode, Pfad ohne Query-String und Statuscode. `user` und `Authorization` erscheinen nicht im Log.
- Keine Speicherung von Nutzer-IDs; `internal/evaluate/evaluate.go` verarbeitet die ID nur transient zur Hash-Berechnung.
- Der In-Memory-Store (`internal/store/store.go`) hält ausschließlich Flag-Daten ohne Personenbezug.

---

## 2. EU Cyber Resilience Act (CRA)

### Befund 2.1 — Kein SBOM / keine dokumentierte Update- und Patch-Strategie sichtbar
- **Schwere:** mittel
- **Fundstelle:** `go.mod` (nur 3 Zeilen, keine Dependencies sichtbar); keine SBOM-Datei im sichtbaren Stand. `SECURITY.md` existiert, Inhalt aber nicht einsehbar und daher nicht bewertbar.
- **Begründung:** Für Produkte mit digitalen Elementen verlangt der CRA dokumentierte Sicherheitseigenschaften, eine Software-Stückliste (SBOM) und einen Prozess für Sicherheitsupdates. Ein reiner Standardbibliotheks-Stack minimiert das Risiko, entbindet aber nicht von der Nachweispflicht.
- **Maßnahme (konkret):**
  - SBOM als Datei einchecken, z. B. `sbom.cdx.json` oder `sbom.spdx.json`, erzeugt mit einem Werkzeug wie Syft oder CycloneDX. Da keine externen Abhängigkeiten sichtbar sind, genügt ein minimales SBOM mit Modul `featureflagservice`, Go-Version und „no external dependencies“.
  - `SECURITY.md` mit mindestens folgenden Abschnitten ergänzen:
    - Security-by-Design-Maßnahmen (Timeouts, Body-Limit, Key-Validierung, Fail-closed-Auth).
    - Update-/Patch-Prozess: Wie wird das Binary aktualisiert? Wer ist verantwortlich?
    - Meldung von Schwachstellen (Kontakt/Email).
    - Bekannte Schwachstellen / Umgang mit CVEs.

### Befund 2.2 — Token-Vergleich nicht konstantzeit
- **Schwere:** mittel
- **Fundstelle:** `internal/middleware/auth.go`, Zeile:
  ```go
  if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != token {
  ```
- **Begründung:** Der Vergleich des Bearer-Tokens mit `!=` ist nicht konstantzeit und kann Timing-Angriffe begünstigen. Der CRA verlangt Security by Design; Geheimnisvergleiche müssen zeitkonstant sein.
- **Maßnahme (konkret):**
  - In `internal/middleware/auth.go` `crypto/subtle.ConstantTimeCompare` verwenden:
    ```go
    import "crypto/subtle"

    presented := strings.TrimPrefix(auth, prefix)
    if subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
        unauthorized(w)
        return
    }
    ```
  - Hinweis: Auch bei leerem `token` den Fail-closed-Pfad beibehalten. Falls Längen unterschiedlich sind, schlägt `ConstantTimeCompare` fehl; das ist hier korrekt.

### Befund 2.3 — Kein Rate-Limiting / Brute-Force-Schutz auf die Authentifizierung
- **Schwere:** niedrig
- **Fundstelle:** `main.go`, `newHandler()`: Es gibt keinen Rate-Limiter vor den geschützten Routen.
- **Begründung:** Ohne Limitierung können Angreifer den Token durch wiederholte Anfragen brute-forcen. Für ein Backend mit statischem Bearer-Token ist das ein realistisches, wenn auch begrenztes Risiko.
- **Maßnahme (konkret):**
  - Optionalen einfachen Rate-Limiter je Client-IP oder pro Token in die Middleware-Kette einbauen (z. B. `golang.org/x/time/rate`; falls keine externen Dependencies gewünscht sind, einen kleinen eigenen Token-Bucket mit Mutex).
  - Mindestens in `SECURITY.md` dokumentieren, dass ein vorgelagerter Reverse Proxy Rate-Limiting übernehmen sollte.

### Befund 2.4 — JSON-Decoder ignoriert überschüssige Daten
- **Schwere:** niedrig
- **Fundstelle:** `internal/api/respond.go`, `decodeJSON`: Nach `dec.Decode(v)` wird nicht geprüft, ob weitere JSON-Daten folgen.
- **Begründung:** Ein Request-Body wie `{...} extra` wird akzeptiert; das verstößt gegen strikte Eingabevalidierung und kann in Kombination mit Proxies zu Desynchronisierung führen.
- **Maßnahme (konkret):**
  - Nach dem ersten Decode prüfen:
    ```go
    if dec.More() {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return errors.New("invalid request body")
    }
    ```
  - Alternativ: `if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) { ... }`.

### Positiv (CRA)
- Server-Timeout-Konfiguration in `main.go` ist vorhanden und geprüft (`ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`).
- Request-Body-Limit (`maxBodyBytes = 1 << 20`) in `internal/api/respond.go` begrenzt Eingaben und antwortet mit `413`.
- Eingabevalidierung der Flag-Keys (`validKey`) verhindert URLs außerhalb des erlaubten Zeichensatzes.
- Fail-closed-Authentifizierung in `internal/middleware/auth.go` schließt geschützte Routen, wenn kein Token gesetzt ist.

---

## 3. EU AI Act

- **Befund:** Keine. Es ist kein KI-System, kein GPAI-Modell und keine automatisierte Entscheidungsfindung im Sinne der KI-VO erkennbar. Der deterministische Hash (`evaluate.Decide`) ist einfache algorithmische Logik ohne KI-Komponente.
- **Maßnahme:** Keine erforderlich.

---

## 4. Pflichttexte & UI (Legal Notice, Terms, Privacy Policy, Cookie/Consent, Impressum)

- **Befund:** Keine unmittelbaren Pflichttexte für ein reines Backend ohne Endnutzer-UI. Es gibt keine Web-Oberfläche, keine Cookies, keinen Verkaufs-/Widerrufskontext.
- **Empfehlung (mittel):** Die API-Dokumentation (`README.md`) sollte einen Datenschutzhinweis für API-Nutzer enthalten (siehe Befund 1.2). Ein Impressum ist nicht erforderlich, sofern kein öffentliches Angebot mit eigener UI betrieben wird.

---

## 5. Barrierefreiheit (WCAG / BITV / EAA)

- **Befund:** Nicht anwendbar. Das Produkt ist eine REST-API ohne öffentliche Web-UI. Es gibt keine HTML-, CSS- oder JavaScript-Oberfläche.
- **Maßnahme:** Keine erforderlich.

---

## 6. Zusammenfassung der offenen Punkte

| # | Schwere | Bereich | Fundstelle | Konkrete Maßnahme |
|---|---------|---------|------------|-------------------|
| 1.1 | hoch | GDPR | `main.go` | TLS-Unterstützung oder verbindliche TLS-Terminierung dokumentieren/erzwingen |
| 1.2 | mittel | GDPR | `README.md` / `COMPLIANCE.md` | Datenschutzabschnitt mit Rechtsgrundlage, Zweck, Speicherdauer, Betroffenenrechten |
| 1.3 | hoch | AC/Marktreife | `internal/api/evaluate.go` | Query-Parameter `user` zusätzlich zu `X-User-ID` verarbeiten |
| 2.1 | mittel | CRA | Repo / `SECURITY.md` | SBOM einchecken und Update-/Patch-Prozess dokumentieren |
| 2.2 | mittel | CRA/Security | `internal/middleware/auth.go` | Konstante Zeit beim Token-Vergleich (`crypto/subtle`) |
| 2.3 | niedrig | CRA/Security | `main.go` / Middleware | Rate-Limiting einbauen oder im Deployment vorschreiben |
| 2.4 | niedrig | CRA/Inputvalidierung | `internal/api/respond.go` | Überschüssige JSON-Daten ablehnen (`dec.More()`) |

**Hinweis:** Die Dateien `COMPLIANCE.md`, `SECURITY.md` und `README.md` existieren laut Dateiliste, ihre Inhalte waren jedoch nicht Teil des sichtbaren Standes. Sollten diese bereits datenschutz- und CRA-relevante Abschnitte enthalten, können einzelne Befunde entfallen; der vorgelegte Code-Stand selbst lässt die oben genannten Lücken jedoch erkennen.