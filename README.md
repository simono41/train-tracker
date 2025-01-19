# Train Tracker

## Beschreibung

Train Tracker ist eine in Go geschriebene Anwendung, die Echtzeitinformationen über Zugbewegungen verfolgt und speichert. Sie nutzt die DB-Vendo-API, um Abfahrtsinformationen von spezifizierten Bahnhöfen abzurufen, berechnet die aktuelle Position der Züge basierend auf ihrer Route und speichert diese Informationen in einer MariaDB-Datenbank.

## Funktionen

- Abrufen von Zugabfahrten für mehrere Bahnhöfe
- Berechnung der aktuellen Zugposition basierend auf Abfahrtszeit und Routeninformationen
- Speichern und Aktualisieren von Zuginformationen in einer MariaDB-Datenbank
- Erfassung und Analyse von Verspätungsdaten
- Automatisches Löschen veralteter Einträge
- Tägliche Übertragung und Aggregation von Verspätungsstatistiken

## Voraussetzungen

- Go 1.15 oder höher
- MariaDB
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

- `DB_DSN`: MariaDB-Datenbankverbindungsstring
- `API_BASE_URL`: Basis-URL der DB-Vendo-API
- `DURATION`: Zeitspanne in Minuten für die Abfrage von Abfahrten
- `DELETE_AFTER_MINUTES`: Zeit in Minuten, nach der alte Einträge gelöscht werden
- `STATION_IDS`: Komma-getrennte Liste von Bahnhofs-IDs
- `UPDATE_INTERVAL_MINUTES`: Intervall in Minuten, in dem der Algorithmus erneut ausgeführt wird (Standard: 1)
- `TRANSFER_TIME`: Uhrzeit für die tägliche Übertragung der Verspätungsstatistiken im Format "HH:MM" (Standard: "23:59")

## Datenbankstruktur

Die Anwendung verwendet drei Haupttabellen:

1. `trips`: Speichert Informationen zu einzelnen Zugfahrten
2. `today_delay_stats`: Speichert tägliche Verspätungsstatistiken
3. `delay_stats`: Speichert aggregierte Verspätungsstatistiken

Detaillierte Tabellenstrukturen finden Sie in der `init.sql` Datei.

## Verwendung

1. Stellen Sie sicher, dass die MariaDB-Datenbank läuft und die Tabellen erstellt wurden.

2. Setzen Sie die erforderlichen Umgebungsvariablen.

3. Starten Sie die Anwendung:
   ```
   go run main.go
   ```

Die Anwendung wird nun kontinuierlich Abfahrtsinformationen abrufen, Zugpositionen berechnen und in der Datenbank speichern.

## Entwicklung

### Code-Struktur

- `main.go`: Hauptanwendungslogik und Einstiegspunkt des Programms
- Funktionen wie `fetchDepartures()`, `fetchTripDetails()`, `savePosition()`, `calculateCurrentPosition()`, `updateTodayDelayStats()`, `transferDailyDelayStats()` und `deleteOldEntries()` implementieren die Kernfunktionalität

### Beitrag

Beiträge sind willkommen! Bitte erstellen Sie einen Pull Request für Verbesserungen oder Fehlerbehebungen.

## Lizenz

Dieses Projekt ist unter der [MIT-Lizenz](https://opensource.org/licenses/MIT) lizenziert.

## Kontakt

Bei Fragen oder Problemen öffnen Sie bitte ein Issue im [Git-Repository](https://code.brothertec.eu/simono41/train-tracker).
