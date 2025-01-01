package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"probe/collector"
	"probe/displayer"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Need to pass in hostname of the database you want to connect to")
		return
	}

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbUser := os.Getenv("DB_USER")
	dbUserPass := os.Getenv("DB_PASSWORD")

	dsn := fmt.Sprintf("%s:%s@tcp(localhost:3306)/", dbUser, dbUserPass)
	db, err := createConnectionPool(dsn, 10, 5, 30*time.Minute)
	if err != nil {
		log.Fatalf("Error initializing connection pool: %v", err)
	}
	defer db.Close()

	metrics := []collector.Metric{
		&collector.OpenConnections{},
		&collector.SlowQueries{},
	}

	stopChan := make(chan struct{})
	collector := collector.NewCollector(db, metrics, stopChan)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		collector.StartCollecting()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		displayer.StartDisplaying(metrics, stopChan)
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	fmt.Println("Shutting down gracefully...")
	close(stopChan)

	wg.Wait()
}

func createConnectionPool(dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Println("Database connection pool initialized")

	return db, nil
}
