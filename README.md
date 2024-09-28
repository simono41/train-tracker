# DB Departure Tracker

## Beschreibung

Dieses Projekt ist ein Go-basierter Service, der Abfahrtsinformationen von verschiedenen Bahnhöfen der Deutschen Bahn abruft und in einer MariaDB-Datenbank speichert. Es verwendet die DB REST API, um Echtzeit-Abfahrtsdaten zu erhalten und aktualisiert die Positionen der Züge in regelmäßigen Abständen.

## Funktionen

- Abruf von Abfahrtsinformationen für mehrere Bahnhöfe
- Konfigurierbare Einstellungen für verschiedene Verkehrsmittel (Bus, Fähre, Straßenbahn, Taxi)
- Speicherung und Aktualisierung von Zugpositionen in einer MariaDB-Datenbank
- Verwendung von UUIDs für eindeutige Datenbankeinträge
- Konfiguration über Umgebungsvariablen

## Voraussetzungen

- Go 1.17 oder höher
- Docker und Docker Compose
- Zugang zur DB REST API

## Installation

1. Klonen Sie das Repository:
   ```
   git clone https://github.com/yourusername/db-departure-tracker.git
   cd db-departure-tracker
   ```

2. Erstellen Sie eine `.env` Datei im Projektverzeichnis und füllen Sie sie mit den erforderlichen Umgebungsvariablen (siehe Konfiguration).

3. Bauen und starten Sie die Docker-Container:
   ```
   docker-compose up --build
   ```

## Konfiguration

Konfigurieren Sie die Anwendung durch Setzen der folgenden Umgebungsvariablen:

- `DB_DSN`: Datenbankverbindungsstring (z.B. "root:password@tcp(mariadb:3306)/traindb")
- `API_BASE_URL`: Basis-URL der DB REST API
- `MAX_RESULTS`: Maximale Anzahl der abzurufenden Ergebnisse pro Anfrage
- `DURATION`: Zeitspanne in Minuten für die Abfrage der Abfahrten
- `BUS`: Einbeziehung von Busabfahrten (true/false)
- `FERRY`: Einbeziehung von Fährabfahrten (true/false)
- `TRAM`: Einbeziehung von Straßenbahnabfahrten (true/false)
- `TAXI`: Einbeziehung von Taxiabfahrten (true/false)
- `STATION_IDS`: Komma-separierte Liste der Bahnhofs-IDs

Beispiel für eine `.env` Datei:

```
DB_DSN=root:password@tcp(mariadb:3306)/traindb
API_BASE_URL=http://db-rest:3000
MAX_RESULTS=10
DURATION=240
BUS=false
FERRY=false
TRAM=false
TAXI=false
STATION_IDS=8000226,8000234
```

## Verwendung

Nach dem Start läuft der Service kontinuierlich und ruft in regelmäßigen Abständen Abfahrtsinformationen ab. Die Daten werden in der konfigurierten MariaDB-Datenbank gespeichert.

## Datenbankschema

Die Anwendung verwendet folgendes Datenbankschema:

```sql
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(36) PRIMARY KEY,
    latitude DOUBLE,
    longitude DOUBLE,
    timestamp DATETIME,
    train_name VARCHAR(50),
    fahrt_nr VARCHAR(20)
);
```

## Entwicklung

Um an diesem Projekt mitzuarbeiten:

1. Forken Sie das Repository
2. Erstellen Sie einen Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Committen Sie Ihre Änderungen (`git commit -m 'Add some AmazingFeature'`)
4. Pushen Sie den Branch (`git push origin feature/AmazingFeature`)
5. Öffnen Sie einen Pull Request

## Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Siehe `LICENSE` Datei für Details.
