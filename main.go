package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type interval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

var db *sql.DB // Database connection pool.

func main() {
	initDb()

	router := gin.Default()
	router.GET("/start", start)
	router.GET("/stop", stop)
	router.GET("/intervals", getIntervals)

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
	time := time.Now()
	_, insErr := tx.Exec("INSERT INTO intervals VALUES (?, ?)", startTime, time)
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
	rows, err := db.Query("SELECT * FROM intervals")
	if err != nil {
		log.Fatal("Unable to execute SELECT query: ", err)
	}
	defer rows.Close()

	var intervals []interval
	for rows.Next() {
		var (
			startTime sql.NullString
			endTime   sql.NullString
			interval  interval
		)
		if err := rows.Scan(&startTime, &endTime); err != nil {
			log.Fatal(err)
		}
		log.Println("start ", startTime, "end", endTime)
		layout := "2006-01-02 15:04:05.999999999Z07:00"
		interval.Start, err = time.Parse(layout, startTime.String)
		if err != nil {
			log.Fatal(err)
		}
		if endTime.String != "" {
			interval.End, err = time.Parse(layout, endTime.String)
		}
		if err != nil {
			log.Fatal(err)
		}
		intervals = append(intervals, interval)
	}
	context.IndentedJSON(http.StatusOK, intervals)
}

func Ping(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
}

func CreateDb() {

	_, err0 := db.Exec(`CREATE TABLE IF NOT EXISTS intervals(
		interval_start timestamp with time zone,
		interval_end timestamp with time zone)`)
	if err0 != nil {
		log.Fatal(err0)
	}

	_, err1 := db.Exec(`CREATE TABLE IF NOT EXISTS active_intervals(
		interval_start timestamp with time zone)`)
	if err1 != nil {
		log.Fatal(err1)
	}
}

func initDb() {
	dsn := "file:gotracked.sqlite?cache=shared&parseTime=true"
	var err error

	// Opening a driver typically will not attempt to connect to the database.
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		// This will not be a connection error, but a DSN parse error or
		// another initialization error.
		log.Fatal("unable to use data source name", err)
	}
	// defer pool.Close()

	db.SetMaxOpenConns(1)

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	appSignal := make(chan os.Signal, 3)
	signal.Notify(appSignal, os.Interrupt)

	go func() {
		select {
		case <-appSignal:
			stop()
		}
	}()

	Ping(ctx)

	CreateDb()
}
