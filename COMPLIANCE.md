VERDICT: CHANGES_REQUESTED

## Prüfgegenstand

Geprüft wurde der im Projektstand sichtbare Go-Backend-Dienst `featureflagservice` (REST-API, Standardbibliothek `net/http`, In-Memory-Store, Logging-Middleware, Tests). Reine Backend-Komponente ohne Endnutzer-UI. Relevante Vorschriften: DSGVO, EU Cyber Resilience Act (CRA). EU AI Act, Impressum/Datenschutzerklärung/Cookie-Pflichten und Barrierefreiheit sind nach dem vorliegenden Projekttyp nicht anwendbar.

---

## 1. DSGVO

### DSGVO-1: Nutzer-ID wird als Query-Parameter übertragen
**Schweregrad:** medium

**Datei:** `internal/api/evaluate.go`

`GET /flags/{key}/evaluate?user={id}` liest die Nutzer-ID aus dem Query-String. Die eigene Logging-Middleware protokolliert zwar ausdrücklich nur `r.URL.Path` und damit keine Query-Parameter (`internal/middleware/logging.go`; Tests bestätigen das). Das erfüllt AC-16/AC-17 im Produkt selbst.

Dennoch verbleibt ein Risiko: Die Nutzer-ID als personenbezogenes Datum steht in der URL und kann in vorgelagerten Systemen (Reverse-Proxy, Load Balancer, CDN, Browser-History, serverseitige Access-Logs Dritter) gespeichert werden. Das betrifft Datenschutz durch Technikgestaltung nach Art. 25 DSGVO.

**Konkrete Abhilfe:**  
Den Evaluate-Endpunkt so ändern, dass die Nutzer-ID nicht im Query-String, sondern in einem Request-Header (z. B. `X-User-ID`) oder in einem POST-Body übertragen wird. Dazu `internal/api/evaluate.go` anpassen und `internal/api/evaluate_test.go` entsprechend umbauen. Falls der Query-Parameter beibehalten werden muss, muss in der Betriebsdokumentation verbindlich festgelegt werden, dass vorgelagerte Systeme Query-Strings nicht loggen dürfen; zusätzlich sollte die App dann prüfen, dass nur HTTPS verwendet wird.

---

### DSGVO-2: Rechtsgrundlage und Datenschutzdokumentation nicht sichtbar
**Schweregrad:** medium

**Datei:** `README.md`

Die App verarbeitet mit `user` eine potenziell personenbezogene Kennung. Der Code speichert sie nicht persistent und protokolliert sie nicht. Trotzdem muss der Verantwortliche für diese Verarbeitung eine Rechtsgrundlage nach Art. 6 DSGVO benennen können und die Verarbeitung dokumentieren. Im sichtbaren Stand fehlt eine solche Dokumentation.

**Konkrete Abhilfe:**  
In `README.md` einen Abschnitt „Datenschutz & Rechtsgrundlage“ ergänzen. Dort ausdrücklich festhalten:

- `user` wird ausschließlich transient zur deterministischen Feature-Rollout-Entscheidung verarbeitet.
- Die Kennung wird nicht persistiert, nicht in App-Logs geschrieben und nicht an Dritte weitergegeben.
- Falls das Backend als Auftragsverarbeiter für einen aufrufenden Dienst betrieben wird, ist ein Auftragsverarbeitungsvertrag erforderlich.
- Der Aufrufer trägt die datenschutzrechtliche Verantwortung gegenüber der betroffenen Person; das Backend selbst stellt keine Benutzeroberfläche und keine Datenschutzerklärung bereit.

---

### DSGVO-3: Löschung und Betroffenenrechte
**Schweregrad:** low

**Datei:** `internal/store/store.go`

Der Store ist ein reiner In-Memory-Store. Personenbezogene Nutzer-IDs werden darin nicht gespeichert. Damit bestehen praktisch keine gespeicherten personenbezogenen Daten, auf die Auskunfts-, Berichtigungs- oder Löschpflichten angewendet werden müssten. Das ist datenschutzfreundlich. Dennoch sollte dieser Umstand dokumentiert werden.

**Konkrete Abhilfe:**  
Im README-Abschnitt „Datenschutz & Rechtsgrundlage“ ausdrücklich ergänzen: „Es werden keine Nutzer-IDs und keine personenbezogenen Daten dauerhaft gespeichert; Betroffenenrechte sind daher auf Anwendungsebene des aufrufenden Dienstes umzusetzen.“

---

## 2. Cyber Resilience Act (CRA)

### CRA-1: Keine Authentifizierung/Autorisierung für Verwaltungsendpunkte
**Schweregrad:** high

**Datei:** `main.go`

Die Routen `POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sind ohne Authentifizierung oder Autorisierung registriert. Jeder, der Netzwerkzugriff auf den Dienst hat, kann Feature-Flags anlegen, verändern oder löschen. Das widerspricht den CRA-Anforderungen an Sicherheit durch Technikgestaltung und sichere Voreinstellungen (Security by design/default), weil der Standardzustand des Produkts ungeschützte Schreibzugriffe erlaubt.

**Konkrete Abhilfe:**  
Eine neue Middleware `internal/middleware/auth.go` ergänzen, die einen konfigurierbaren API-Token/Bearer-Token aus einer Umgebungsvariable (z. B. `API_TOKEN`) prüft. Mindestens alle schreibenden Routen (`POST`, `PUT`, `DELETE`) schützen; empfehlenswert ist auch der Schutz der Leserouten. In `main.go` die Middleware vor die Handler schalten, Tests in `main_test.go` ergänzen und in `README.md` einen Abschnitt „Security / Authentication“ aufnehmen.

---

### CRA-2: Keine Transportverschlüsselung und Standardbindung an alle Interfaces
**Schweregrad:** high

**Datei:** `main.go`

Der Server wird über `http.Server` ohne TLS gestartet. Die Standardadresse ist `:8080` und bindet damit an alle verfügbaren Netzwerkinterfaces. Da die Nutzer-ID über den Query-String `?user=...` übermittelt wird, kann sie bei direkter Exposition unverschlüsselt über das Netzwerk übertragen werden. Das ist weder datenschutzfreundlich noch CRA-konform.

**Konkrete Abhilfe:**  
Die Standardadresse in `main.go` von `:8080` auf `127.0.0.1:8080` ändern, damit der Dienst nicht versehentlich öffentlich lauscht. Zusätzlich entweder TLS direkt in `main.go` implementieren (`ListenAndServeTLS`) oder verbindlich in `README.md` vorschreiben, dass der Betrieb ausschließlich hinter einem TLS-terminierenden Reverse-Proxy erfolgt. Die erlaubten Betriebsarten müssen dokumentiert werden.

---

### CRA-3: Sicherheitsdokumentation und SBOM nicht sichtbar
**Schweregrad:** medium

**Datei:** `README.md`, `go.mod`

Der Code enthält gute technische Sicherheitsmaßnahmen (Server-Timeouts, Body-Limit, Key-Validierung, keine PII in Logs). Eine CRA-konforme Sicherheitsdokumentation und ein Software Bill of Materials (SBOM) sind im sichtbaren Stand jedoch nicht belegt. `go.mod` existiert, der vollständige Inhalt ist aber nur mit drei Zeilen angegeben; es ist nicht erkennbar, ob ein SBOM oder eine Abhängigkeitsliste dokumentiert ist.

**Konkrete Abhilfe:**  
`README.md` um einen Abschnitt „Security“ ergänzen, der die umgesetzten Sicherheitseigenschaften nennt:

- Server-Timeouts (`ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`)
- Body-Limit von 1 MiB für schreibende Endpunkte
- Validierung der Flag-Keys
- Logging ohne Query-String und ohne Nutzer-IDs
- Authentifizierungs- und TLS-Konzept nach Umsetzung der CRA-1/CRA-2-Maßnahmen

Zusätzlich einen SBOM-Prozess einführen: `go.mod`/`go.sum` als Abhängigkeitsnachweis bestätigen oder ein SBOM im CycloneDX-/SPDX-Format erzeugen. Einen Update-/Patch-Prozess dokumentieren (z. B. Release- und Patch-Bereitstellung, Umgang mit bekannten Schwachstellen).

---

## 3. EU AI Act

Keine KI-Funktion im sichtbaren Code. Der Dienst führt deterministische Hash-Entscheidungen aus. Es bestehen keine Pflichten nach EU AI Act.

---

## 4. Pflichttexte & UI

Nicht anwendbar. Das Produkt ist ein reines REST-Backend ohne Endnutzer-UI. Es gibt keine Impressums-, Datenschutzerklärungs-, Cookie-Banner- oder Widerrufsbelehrungspflichten auf dieser Ebene. Solche Texte muss die aufrufende Anwendung bereitstellen, nicht dieses Backend.

---

## 5. Barrierefreiheit

Nicht anwendbar. Es existiert keine öffentliche Web-UI. Die API selbst unterliegt nicht den WCAG-/BITV-/EAA-Anforderungen.

---

## Gesamtbewertung

Der Dienst erfüllt die spezifizierten funktionalen Anforderungen und die datenschutzfreundliche Logging-Vorgabe weitgehend. Es gibt keine Hinweise auf einen fundamentalen DSGVO-Verstoß wie persistente Speicherung oder Protokollierung personenbezogener Daten im Klartext. Die festgestellten Mängel sind behebbar: fehlende Authentifizierung, fehlende Transportverschlüsselung/unsichere Standardbindung und fehlende CRA-Sicherheitsdokumentation. Daher: `CHANGES_REQUESTED`.