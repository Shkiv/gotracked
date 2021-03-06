package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type interval struct {
	UUID  uuid.UUID `json:"uuid"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

var (
	db     *sql.DB
	layout string = "2006-01-02 15:04:05.999999999Z07:00"
)

func main() {
	dsn := "file:gotracked.sqlite?cache=shared&parseTime=true"
	var err error

	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal("unable to use data source name", err)
	}
	defer db.Close()

	CreateDb()

	router := gin.Default()
	router.GET("/start", start)
	router.GET("/stop", stop)
	router.GET("/intervals", getIntervals)
	router.GET("/active-interval", getActiveInterval)

	router.Run("localhost:8090")
}

func start(context *gin.Context) {
	rows := db.QueryRow("SELECT COUNT(*) FROM active_intervals")
	var count int
	rows.Scan(&count)
	if count < 1 {
		time := time.Now()
		_, err := db.Exec("INSERT INTO active_intervals (interval_start) VALUES (?)", time)
		if err != nil {
			log.Print(err)
			context.JSON(http.StatusInternalServerError, nil)
			return
		}
		context.JSON(http.StatusCreated, "Created")
		return
	}
	context.JSON(http.StatusOK, "Not changed")
}

func stop(context *gin.Context) {
	var startTime sql.NullString

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	scanErr := tx.QueryRow("SELECT interval_start FROM active_intervals").Scan(&startTime)
	if scanErr != nil {
		tx.Rollback()
		context.JSON(http.StatusOK, "No interval")
		return
	}
	_, insErr := tx.Exec("INSERT INTO intervals (uuid, interval_start, interval_end) VALUES (?, ?, ?)", uuid.New(), startTime, time.Now())
	if insErr != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, nil)
		return
	}
	_, delErr := tx.Exec("DELETE FROM active_intervals")
	if delErr != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, nil)
		return
	}
	tx.Commit()

	context.JSON(http.StatusOK, "OK")
}

func getIntervals(context *gin.Context) {
	rows, err := db.Query("SELECT uuid, interval_start, interval_end FROM intervals")
	if err != nil {
		log.Fatal("Unable to execute SELECT query: ", err)
	}
	defer rows.Close()

	var intervals []interval
	for rows.Next() {
		var (
			uuid      uuid.UUID
			startTime sql.NullString
			endTime   sql.NullString
			interval  interval
		)
		if err := rows.Scan(&uuid, &startTime, &endTime); err != nil {
			log.Print(err)
		}
		interval.UUID = uuid
		interval.Start, err = time.Parse(layout, startTime.String)
		if err != nil {
			log.Print(err)
		}
		if endTime.Valid {
			interval.End, err = time.Parse(layout, endTime.String)
		}
		if err != nil {
			log.Print(err)
		}
		intervals = append(intervals, interval)
	}
	context.IndentedJSON(http.StatusOK, intervals)
}

func getActiveInterval(context *gin.Context) {
	var startTimeString sql.NullString
	db.QueryRow("SELECT interval_start FROM active_intervals").Scan(&startTimeString)
	startTime, err := time.Parse(layout, startTimeString.String)
	if err != nil {
		context.JSON(http.StatusOK, nil)
		return
	}
	context.JSON(http.StatusOK, startTime)
}

func CreateDb() {

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS intervals(
		uuid blob,
		interval_start timestamp with time zone,
		interval_end timestamp with time zone)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS active_intervals(
		interval_start timestamp with time zone)`)
	if err != nil {
		log.Fatal(err)
	}
}

func UpdateDb() {
	_, err := db.Exec(`ALTER TABLE intervals
		ADD uuid blob
	`)
	if err != nil {
		log.Fatal(err)
	}
}
