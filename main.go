package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
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

    for {
        for _, stationID := range stationIDs {
            departures := fetchDepartures(apiBaseURL, stationID, duration)
            for _, dep := range departures {
                savePosition(db, dep)
            }
        }
        deleteOldEntries(db, deleteAfter)
        time.Sleep(1 * time.Minute)
    }
}

func fetchDepartures(apiBaseURL, stationID string, duration int) []Departure {
    url := fmt.Sprintf("%s/stops/%s/departures?duration=%d&linesOfStops=false&remarks=false&language=de&nationalExpress=true&national=true&regionalExpress=true&regional=true&suburban=true&bus=false&ferry=false&subway=false&tram=false&taxi=false&pretty=true",
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

func savePosition(db *sql.DB, dep Departure) {
    whenTime, err := time.Parse(time.RFC3339, dep.When)
    if err != nil {
        log.Printf("Fehler beim Parsen der Zeit: %v\n", err)
        return
    }

    today := whenTime.Format("2006-01-02")

    var existingID string
    err = db.QueryRow("SELECT id FROM trips WHERE fahrt_nr = ? AND DATE(timestamp) = ?", dep.Line.FahrtNr, today).Scan(&existingID)

    if err == sql.ErrNoRows {
        id := uuid.New().String()
        _, err = db.Exec("INSERT INTO trips (id, timestamp, train_name, fahrt_nr, trip_id) VALUES (?, ?, ?, ?, ?)",
            id, whenTime, dep.Line.Name, dep.Line.FahrtNr, dep.TripId)
        if err != nil {
            log.Printf("Fehler beim Speichern der neuen Position: %v\n", err)
        } else {
            log.Printf("Neue Position gespeichert (ID: %s, Zug: %s, FahrtNr: %s)\n", id, dep.Line.Name, dep.Line.FahrtNr)
        }
    } else if err == nil {
        _, err = db.Exec("UPDATE trips SET timestamp = ?, train_name = ?, trip_id = ? WHERE id = ?",
            whenTime, dep.Line.Name, dep.TripId, existingID)
        if err != nil {
            log.Printf("Fehler beim Aktualisieren der Position: %v\n", err)
        } else {
            log.Printf("Position aktualisiert (ID: %s, Zug: %s, FahrtNr: %s)\n", existingID, dep.Line.Name, dep.Line.FahrtNr)
        }
    } else {
        log.Printf("Fehler bei der Überprüfung des existierenden Eintrags: %v\n", err)
    }
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
