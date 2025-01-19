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
    TripId string `json:"tripId"`
    When   string `json:"when"`
    Line   struct {
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

    db, err := sql.Open("mysql", dbDSN)
    if err != nil {
        log.Fatal("Fehler beim Verbinden mit der Datenbank: ", err)
    }
    defer db.Close()

    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        for _, stationID := range stationIDs {
            departures := fetchDepartures(apiBaseURL, stationID, duration)
            for _, dep := range departures {
                savePosition(db, dep, apiBaseURL)
            }
        }
        deleteOldEntries(db, deleteAfter)
        
        select {
        case <-ticker.C:
            logDatabaseStats(db)
        default:
            // Do nothing
        }
        
        time.Sleep(1 * time.Minute)
    }
}

func fetchDepartures(apiBaseURL, stationID string, duration int) []Departure {
    url := fmt.Sprintf("%s/stops/%s/departures?duration=%d&linesOfStops=false&remarks=true&language=en&nationalExpress=true&national=true&regionalExpress=true&regional=true&suburban=true&bus=false&ferry=false&subway=false&tram=false&taxi=false&pretty=true",
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
    url := fmt.Sprintf("%s/trips/%s?stopovers=true&remarks=true&polyline=true&language=en", apiBaseURL, escapedTripID)
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
    tripDetails, err := fetchTripDetails(apiBaseURL, dep.TripId)
    if err != nil {
        log.Printf("Fehler beim Abrufen der Zugdetails für TripID %s: %v\n", dep.TripId, err)
        return
    }

    currentTime := time.Now()
    longitude, latitude := calculateCurrentPosition(tripDetails, currentTime)

    whenTime, err := time.Parse(time.RFC3339, dep.When)
    if err != nil {
        log.Printf("Fehler beim Parsen der Zeit für TripID %s: %v\n", dep.TripId, err)
        return
    }

    today := whenTime.Format("2006-01-02")

    var existingID string
    err = db.QueryRow("SELECT id FROM trips WHERE fahrt_nr = ? AND DATE(timestamp) = ?", dep.Line.FahrtNr, today).Scan(&existingID)

    if err == sql.ErrNoRows {
        id := uuid.New().String()
        _, err = db.Exec("INSERT INTO trips (id, timestamp, train_name, fahrt_nr, trip_id, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?, ?)",
            id, whenTime, dep.Line.Name, dep.Line.FahrtNr, dep.TripId, latitude, longitude)
        if err != nil {
            log.Printf("Fehler beim Speichern der neuen Position für TripID %s: %v\n", dep.TripId, err)
        } else {
            log.Printf("Neue Position gespeichert (ID: %s, Zug: %s, FahrtNr: %s, Lat: %f, Lon: %f)\n", id, dep.Line.Name, dep.Line.FahrtNr, latitude, longitude)
        }
    } else if err == nil {
        _, err = db.Exec("UPDATE trips SET timestamp = ?, train_name = ?, trip_id = ?, latitude = ?, longitude = ? WHERE id = ?",
            whenTime, dep.Line.Name, dep.TripId, latitude, longitude, existingID)
        if err != nil {
            log.Printf("Fehler beim Aktualisieren der Position für TripID %s: %v\n", dep.TripId, err)
        } else {
            log.Printf("Position aktualisiert (ID: %s, Zug: %s, FahrtNr: %s, Lat: %f, Lon: %f)\n", existingID, dep.Line.Name, dep.Line.FahrtNr, latitude, longitude)
        }
    } else {
        log.Printf("Fehler bei der Überprüfung des existierenden Eintrags für TripID %s: %v\n", dep.TripId, err)
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
