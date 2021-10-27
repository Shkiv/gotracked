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
	router.GET("/end", end)
	router.GET("/intervals", getIntervals)

	router.Run("localhost:8080")
}

func start(context *gin.Context) {
	context.JSON(http.StatusCreated, "OK")

	rows := db.QueryRow("SELECT count(*) FROM active_intervals")
	var count int
	rows.Scan(&count)
	if count < 1 {
		time := time.Now()
		_, err := db.Exec("INSERT INTO active_intervals VALUES (?, null)", time)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func end(context *gin.Context) {

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
		interval_start timestamp with time zone,
		interval_end timestamp with time zone)`)
	if err1 != nil {
		log.Fatal(err0)
	}
}

func initDb() {
	dsn := "file:locked.sqlite?cache=shared&parseTime=true"
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
