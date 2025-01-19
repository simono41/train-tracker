package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "math"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"
)

type Departure struct {
    TripId         string    `json:"tripId"`
    When           string    `json:"when"`
    PlannedWhen    string    `json:"plannedWhen"`
    Delay          int       `json:"delay"`
    Line           struct {
        Name    string `json:"name"`
        FahrtNr string `json:"fahrtNr"`
    } `json:"line"`
}

type APIResponse struct {
    Departures []Departure `json:"departures"`
}

type TripDetails struct {
    Origin      Station   `json:"origin"`
    Destination Station   `json:"destination"`
    Departure   time.Time `json:"departure"`
    Arrival     time.Time `json:"arrival"`
    Polyline    Polyline  `json:"polyline"`
}

type Station struct {
    Name     string   `json:"name"`
    Location Location `json:"location"`
}

type Location struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type Polyline struct {
    Features []Feature `json:"features"`
}

type Feature struct {
    Geometry Geometry `json:"geometry"`
}

type Geometry struct {
    Coordinates []float64 `json:"coordinates"`
}

func main() {
    log.SetOutput(os.Stdout)
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
    log.Println("Anwendung gestartet")

    dbDSN := os.Getenv("DB_DSN")
    apiBaseURL := os.Getenv("API_BASE_URL")
    duration, err := strconv.Atoi(os.Getenv("DURATION"))
    if err != nil {
        log.Fatalf("Ungültiger Wert für DURATION: %v", err)
    }
    deleteAfter, err := strconv.Atoi(os.Getenv("DELETE_AFTER_MINUTES"))
    if err != nil {
        log.Fatalf("Ungültiger Wert für DELETE_AFTER_MINUTES: %v", err)
    }
    stationIDs := strings.Split(os.Getenv("STATION_IDS"), ",")

    updateInterval, err := strconv.Atoi(os.Getenv("UPDATE_INTERVAL_MINUTES"))
    if err != nil || updateInterval <= 0 {
        log.Println("Ungültiger oder fehlender Wert für UPDATE_INTERVAL_MINUTES, verwende Standardwert von 1 Minute")
        updateInterval = 1
    }

    transferTimeStr := os.Getenv("TRANSFER_TIME")
    transferTime, err := time.Parse("15:04", transferTimeStr)
    if err != nil {
        log.Printf("Ungültiger Wert für TRANSFER_TIME, verwende Standardwert 23:00")
        transferTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 23, 0, 0, 0, time.Local)
    } else {
        now := time.Now()
        transferTime = time.Date(now.Year(), now.Month(), now.Day(), transferTime.Hour(), transferTime.Minute(), 0, 0, time.Local)
    }

    db, err := sql.Open("mysql", dbDSN)
    if err != nil {
        log.Fatal("Fehler beim Verbinden mit der Datenbank: ", err)
    }
    defer db.Close()

    ticker := time.NewTicker(5 * time.Minute)
    updateTicker := time.NewTicker(time.Duration(updateInterval) * time.Minute)
    defer ticker.Stop()
    defer updateTicker.Stop()

    for {
        select {
        case <-updateTicker.C:
            for _, stationID := range stationIDs {
                departures := fetchDepartures(apiBaseURL, stationID, duration)
                // Füge einen 1-Sekunden-Sleeper hinzu
                time.Sleep(1 * time.Second)
                for _, dep := range departures {
                    savePosition(db, dep, apiBaseURL)
                }
            }
            deleteOldEntries(db, deleteAfter)
        case <-ticker.C:
            logDatabaseStats(db)
        default:
            now := time.Now()
            if now.After(transferTime) {
                transferDailyDelayStats(db)
                transferTime = time.Date(now.Year(), now.Month(), now.Day()+1, transferTime.Hour(), transferTime.Minute(), 0, 0, time.Local)
            }
            time.Sleep(1 * time.Minute)
        }
    }
}

func fetchDepartures(apiBaseURL, stationID string, duration int) []Departure {
    url := fmt.Sprintf("%s/stops/%s/departures?duration=%d&linesOfStops=false&remarks=true&language=en&nationalExpress=true&national=true&regionalExpress=true&regional=true&suburban=true&bus=false&ferry=false&subway=false&tram=false&taxi=false&pretty=false",
        apiBaseURL, stationID, duration)
    resp, err := http.Get(url)
    if err != nil {
        log.Printf("Fehler beim Abrufen der Abfahrten für Station %s: %v\n", stationID, err)
        return nil
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Fehler beim Lesen der Antwort für Station %s: %v\n", stationID, err)
        return nil
    }

    if len(body) == 0 {
        log.Printf("Leere Antwort vom Server für Station %s erhalten\n", stationID)
        return nil
    }

    var response APIResponse
    err = json.Unmarshal(body, &response)
    if err != nil {
        log.Printf("Fehler beim Dekodieren der Abfahrten für Station %s: %v\nAntwort-Body: %s\n", stationID, err, string(body))
        return nil
    }

    log.Printf("Erfolgreich %d Abfahrten für Station %s abgerufen\n", len(response.Departures), stationID)
    return response.Departures
}

func fetchTripDetails(apiBaseURL, tripID string) (*TripDetails, error) {
    escapedTripID := url.QueryEscape(tripID)
    url := fmt.Sprintf("%s/trips/%s?stopovers=true&remarks=true&polyline=true&language=en&pretty=false", apiBaseURL, escapedTripID)
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("Fehler beim Abrufen der Zugdetails: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Fehler beim Lesen der Antwort: %v", err)
    }

    if len(body) == 0 {
        return nil, fmt.Errorf("Leere Antwort vom Server erhalten")
    }

    var tripResponse struct {
        Trip TripDetails `json:"trip"`
    }
    if err := json.Unmarshal(body, &tripResponse); err != nil {
        return nil, fmt.Errorf("Fehler beim Dekodieren der Zugdetails: %v", err)
    }

    if tripResponse.Trip.Origin.Name == "" || tripResponse.Trip.Destination.Name == "" {
        return nil, fmt.Errorf("Unvollständige Tripdaten erhalten")
    }

    return &tripResponse.Trip, nil
}

func savePosition(db *sql.DB, dep Departure, apiBaseURL string) {
    // Füge einen 1-Sekunden-Sleeper hinzu
    time.Sleep(1 * time.Second)

    tripDetails, err := fetchTripDetails(apiBaseURL, dep.TripId)
    if err != nil {
        log.Printf("Fehler beim Abrufen der Zugdetails für TripID %s: %v\n", dep.TripId, err)
        return
    }

    currentTime := time.Now()
    longitude, latitude := calculateCurrentPosition(tripDetails, currentTime)

    if dep.When == "" {
        log.Printf("Warnung: Leerer Zeitstempel für FahrtNr %s, überspringe Eintrag\n", dep.Line.FahrtNr)
        return
    }

    whenTime, err := time.Parse(time.RFC3339, dep.When)
    if err != nil {
        log.Printf("Fehler beim Parsen der Zeit für TripID %s: %v\n", dep.TripId, err)
        return
    }

    plannedWhenTime, err := time.Parse(time.RFC3339, dep.PlannedWhen)
    if err != nil {
        log.Printf("Fehler beim Parsen der geplanten Zeit für TripID %s: %v\n", dep.TripId, err)
        return
    }

    today := whenTime.Format("2006-01-02")

    var existingID string
    err = db.QueryRow("SELECT id FROM trips WHERE fahrt_nr = ? AND DATE(timestamp) = ?", dep.Line.FahrtNr, today).Scan(&existingID)

    if err == sql.ErrNoRows {
        id := uuid.New().String()
        _, err = db.Exec("INSERT INTO trips (id, timestamp, planned_timestamp, delay, train_name, fahrt_nr, trip_id, latitude, longitude, destination) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            id, whenTime, plannedWhenTime, dep.Delay, dep.Line.Name, dep.Line.FahrtNr, dep.TripId, latitude, longitude, tripDetails.Destination.Name)
        if err != nil {
            log.Printf("Fehler beim Speichern der neuen Position für TripID %s: %v\n", dep.TripId, err)
        } else {
            log.Printf("Neue Position gespeichert (ID: %s, Zug: %s, FahrtNr: %s, Lat: %f, Lon: %f, Verspätung: %d, Ziel: %s)\n", id, dep.Line.Name, dep.Line.FahrtNr, latitude, longitude, dep.Delay, tripDetails.Destination.Name)
        }
    } else if err == nil {
        _, err = db.Exec("UPDATE trips SET timestamp = ?, planned_timestamp = ?, delay = ?, train_name = ?, trip_id = ?, latitude = ?, longitude = ?, destination = ? WHERE id = ?",
            whenTime, plannedWhenTime, dep.Delay, dep.Line.Name, dep.TripId, latitude, longitude, tripDetails.Destination.Name, existingID)
        if err != nil {
            log.Printf("Fehler beim Aktualisieren der Position für TripID %s: %v\n", dep.TripId, err)
        } else {
            log.Printf("Position aktualisiert (ID: %s, Zug: %s, FahrtNr: %s, Lat: %f, Lon: %f, Verspätung: %d, Ziel: %s)\n", existingID, dep.Line.Name, dep.Line.FahrtNr, latitude, longitude, dep.Delay, tripDetails.Destination.Name)
        }
    } else {
        log.Printf("Fehler bei der Überprüfung des existierenden Eintrags für TripID %s: %v\n", dep.TripId, err)
    }

    updateTodayDelayStats(db, dep.Line.FahrtNr, dep.Line.Name, dep.Delay, whenTime)
}

func updateTodayDelayStats(db *sql.DB, fahrtNr, trainName string, delay int, timestamp time.Time) {
    var existingID string
    err := db.QueryRow("SELECT id FROM today_delay_stats WHERE fahrt_nr = ? AND DATE(timestamp) = CURDATE()", fahrtNr).Scan(&existingID)

    if err == sql.ErrNoRows {
        // Kein existierender Eintrag, führe INSERT aus
        _, err = db.Exec(`
            INSERT INTO today_delay_stats (id, fahrt_nr, train_name, delay, timestamp)
            VALUES (UUID(), ?, ?, ?, ?)
        `, fahrtNr, trainName, delay, timestamp)
        if err != nil {
            log.Printf("Fehler beim Einfügen der heutigen Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
        }
    } else if err == nil {
        // Existierender Eintrag gefunden, führe UPDATE aus
        _, err = db.Exec(`
            UPDATE today_delay_stats
            SET train_name = ?, delay = ?, timestamp = ?
            WHERE id = ?
        `, trainName, delay, timestamp, existingID)
        if err != nil {
            log.Printf("Fehler beim Aktualisieren der heutigen Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
        }
    } else {
        log.Printf("Fehler beim Überprüfen der heutigen Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
    }
}

func transferDailyDelayStats(db *sql.DB) {
    rows, err := db.Query("SELECT fahrt_nr, train_name, delay FROM today_delay_stats WHERE DATE(timestamp) = CURDATE()")
    if err != nil {
        log.Printf("Fehler beim Abrufen der heutigen Verspätungsstatistiken: %v\n", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var fahrtNr, trainName string
        var delay int
        if err := rows.Scan(&fahrtNr, &trainName, &delay); err != nil {
            log.Printf("Fehler beim Scannen der Verspätungsdaten: %v\n", err)
            continue
        }

        var existingID string
        err := db.QueryRow("SELECT id FROM delay_stats WHERE fahrt_nr = ?", fahrtNr).Scan(&existingID)

        if err == sql.ErrNoRows {
            // Kein existierender Eintrag, führe INSERT aus
            _, err = db.Exec(`
                INSERT INTO delay_stats (id, fahrt_nr, total_trips, delayed_trips, avg_delay, last_updated)
                VALUES (UUID(), ?, 1, ?, ?, NOW())
            `, fahrtNr, delay > 300, delay)
            if err != nil {
                log.Printf("Fehler beim Einfügen der Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
            }
        } else if err == nil {
            // Existierender Eintrag gefunden, führe UPDATE aus
            _, err = db.Exec(`
                UPDATE delay_stats
                SET total_trips = total_trips + 1,
                    delayed_trips = delayed_trips + ?,
                    avg_delay = ((avg_delay * total_trips) + ?) / (total_trips + 1),
                    last_updated = NOW()
                WHERE id = ?
            `, delay > 300, delay, existingID)
            if err != nil {
                log.Printf("Fehler beim Aktualisieren der Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
            }
        } else {
            log.Printf("Fehler beim Überprüfen der Verspätungsstatistiken für FahrtNr %s: %v\n", fahrtNr, err)
        }
    }

    // Löschen Sie die heutigen Statistiken nach der Übertragung
    _, err = db.Exec("DELETE FROM today_delay_stats WHERE DATE(timestamp) = CURDATE()")
    if err != nil {
        log.Printf("Fehler beim Löschen der heutigen Verspätungsstatistiken: %v\n", err)
    }
}

func calculateCurrentPosition(trip *TripDetails, currentTime time.Time) (float64, float64) {
    totalDuration := trip.Arrival.Sub(trip.Departure)
    elapsedDuration := currentTime.Sub(trip.Departure)
    progress := elapsedDuration.Seconds() / totalDuration.Seconds()

    if progress < 0 {
        return trip.Origin.Location.Longitude, trip.Origin.Location.Latitude
    }
    if progress > 1 {
        return trip.Destination.Location.Longitude, trip.Destination.Location.Latitude
    }

    polyline := trip.Polyline.Features
    totalDistance := 0.0
    distances := make([]float64, len(polyline)-1)

    for i := 0; i < len(polyline)-1; i++ {
        dist := distance(
            polyline[i].Geometry.Coordinates[1], polyline[i].Geometry.Coordinates[0],
            polyline[i+1].Geometry.Coordinates[1], polyline[i+1].Geometry.Coordinates[0],
        )
        distances[i] = dist
        totalDistance += dist
    }

    targetDistance := totalDistance * progress
    coveredDistance := 0.0

    for i, dist := range distances {
        if coveredDistance+dist > targetDistance {
            remainingDistance := targetDistance - coveredDistance
            ratio := remainingDistance / dist
            return interpolate(
                polyline[i].Geometry.Coordinates[0], polyline[i].Geometry.Coordinates[1],
                polyline[i+1].Geometry.Coordinates[0], polyline[i+1].Geometry.Coordinates[1],
                ratio,
            )
        }
        coveredDistance += dist
    }

    return trip.Destination.Location.Longitude, trip.Destination.Location.Latitude
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
    const r = 6371 // Earth radius in kilometers

    dLat := (lat2 - lat1) * math.Pi / 180
    dLon := (lon2 - lon1) * math.Pi / 180
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
            math.Sin(dLon/2)*math.Sin(dLon/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    return r * c
}

func interpolate(lon1, lat1, lon2, lat2, ratio float64) (float64, float64) {
    return lon1 + (lon2-lon1)*ratio, lat1 + (lat2-lat1)*ratio
}

func deleteOldEntries(db *sql.DB, deleteAfterMinutes int) {
    deleteTime := time.Now().Add(time.Duration(-deleteAfterMinutes) * time.Minute)
    result, err := db.Exec("DELETE FROM trips WHERE timestamp < ?", deleteTime)
    if err != nil {
        log.Printf("Fehler beim Löschen alter Einträge: %v\n", err)
        return
    }
    rowsAffected, _ := result.RowsAffected()
    log.Printf("%d alte Einträge gelöscht\n", rowsAffected)
}

func logDatabaseStats(db *sql.DB) {
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM trips").Scan(&count)
    if err != nil {
        log.Printf("Fehler beim Abrufen der Datenbankstatistiken: %v\n", err)
        return
    }
    log.Printf("Aktuelle Anzahl der Einträge in der Datenbank: %d\n", count)
}

