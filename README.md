# Train Tracker

## Beschreibung

Train Tracker ist ein in Go geschriebenes Programm, das Echtzeitinformationen über Zugbewegungen verfolgt und speichert. Es nutzt die DB-Vendo-API, um Abfahrtsinformationen von spezifizierten Bahnhöfen abzurufen, berechnet die aktuelle Position der Züge basierend auf ihrer Route und speichert diese Informationen in einer MySQL-Datenbank.

## Funktionen

- Abrufen von Zugabfahrten für mehrere Bahnhöfe
- Berechnung der aktuellen Zugposition basierend auf Abfahrtszeit und Routeninformationen
- Speichern und Aktualisieren von Zuginformationen in einer MySQL-Datenbank
- Automatisches Löschen veralteter Einträge
- Regelmäßige Protokollierung von Datenbankstatistiken

## Voraussetzungen

- Go 1.15 oder höher
- MySQL-Datenbank
- Zugang zur DB-Vendo-API

## Installation

1. Klonen Sie das Repository:
   ```
   git clone https://code.brothertec.eu/simono41/train-tracker.git
   ```

2. Navigieren Sie in das Projektverzeichnis:
   ```
   cd train-tracker
   ```

3. Installieren Sie die Abhängigkeiten:
   ```
   go mod tidy
   ```

## Konfiguration

Konfigurieren Sie die Anwendung über folgende Umgebungsvariablen:

- `DB_DSN`: MySQL-Datenbankverbindungsstring
- `API_BASE_URL`: Basis-URL der DB-Vendo-API
- `DURATION`: Zeitspanne in Minuten für die Abfrage von Abfahrten
- `DELETE_AFTER_MINUTES`: Zeit in Minuten, nach der alte Einträge gelöscht werden
- `STATION_IDS`: Komma-getrennte Liste von Bahnhofs-IDs

## Datenbankstruktur

Stellen Sie sicher, dass Ihre MySQL-Datenbank eine `trips`-Tabelle mit folgender Struktur hat:

```sql
CREATE TABLE trips (
    id VARCHAR(36) PRIMARY KEY,
    timestamp DATETIME,
    train_name VARCHAR(255),
    fahrt_nr VARCHAR(255),
    trip_id VARCHAR(255),
    latitude DOUBLE,
    longitude DOUBLE
);
```

## Verwendung

1. Setzen Sie die erforderlichen Umgebungsvariablen.

2. Starten Sie die Anwendung:
   ```
   go run main.go
   ```

Die Anwendung wird nun kontinuierlich Abfahrtsinformationen abrufen, die Zugpositionen berechnen und in der Datenbank speichern.

## Entwicklung

### Code-Struktur

- `main.go`: Hauptanwendungslogik und Einstiegspunkt des Programms
- Funktionen wie `fetchDepartures()`, `fetchTripDetails()`, `savePosition()`, `calculateCurrentPosition()` und `deleteOldEntries()` implementieren die Kernfunktionalität

### Beitrag

Beiträge sind willkommen! Bitte erstellen Sie einen Pull Request für Verbesserungen oder Fehlerbehebungen.

## Lizenz

Dieses Projekt ist unter der [MIT-Lizenz](https://opensource.org/licenses/MIT) lizenziert.

## Kontakt

Bei Fragen oder Problemen öffnen Sie bitte ein Issue im [Git-Repository](https://code.brothertec.eu/simono41/train-tracker).
