package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	logFilePath    = "stress_test.log"
	apiBaseURL     = "http://localhost:8080/api/v1"
	dbConnString   = "postgres://postgres:postgres@localhost:5432/pusingberat?sslmode=disable"
	linesPerSecond = 1000
	duration       = 30 * time.Second
)

type LogSource struct {
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
	LogType  string `json:"log_type"`
}

func main() {
	log.Println("Starting PUSINGBERAT Stress Test")

	// 1. Create a dummy log file
	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Clean up existing file if any
	os.Remove(absPath)
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer file.Close()

	// 2. Connect to the database to verify counts later
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbConnString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	var beforeCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&beforeCount)
	if err != nil {
		log.Fatalf("Failed to query initial event count: %v", err)
	}

	// 3. Register log source with SIEM via API
	source := LogSource{
		Name:     "Stress Test Source",
		FilePath: absPath,
		LogType:  "generic",
	}
	payload, _ := json.Marshal(source)

	req, _ := http.NewRequest("POST", apiBaseURL+"/sources", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to register log source: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Printf("Warning: API returned status %d. Ensure the SIEM is running and log source doesn't already exist.", resp.StatusCode)
	}

	// 4. Blast logs
	log.Printf("Blasting logs at %d lines/sec for %v...", linesPerSecond, duration)
	
	// Create a channel to signal completion of writing
	done := make(chan bool)
	
	totalSent := 0
	
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(linesPerSecond))
		defer ticker.Stop()

		for i := 0; i < linesPerSecond * int(duration.Seconds()); i++ {
			<-ticker.C
			timestamp := time.Now().Format(time.RFC3339)
			logLine := fmt.Sprintf("[%s] INFO Stress test event number %d\n", timestamp, i)
			if _, err := file.WriteString(logLine); err != nil {
				log.Printf("Failed to write line: %v", err)
			}
			totalSent++
		}
		done <- true
	}()

	<-done
	log.Printf("Finished writing %d lines. Waiting for SIEM to process...", totalSent)
	
	// 5. Wait for SIEM to ingest everything
	time.Sleep(5 * time.Second)

	// 6. Verify 100% ingested
	var afterCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&afterCount)
	if err != nil {
		log.Fatalf("Failed to query final event count: %v", err)
	}

	ingested := int(afterCount - beforeCount)
	log.Printf("Total Sent: %d, Total Ingested: %d", totalSent, ingested)

	if ingested >= totalSent {
		log.Println("SUCCESS: 100% of events ingested without dropping!")
	} else {
		log.Fatalf("FAILURE: Dropped %d events. (Ingested %d / %d)", totalSent-ingested, ingested, totalSent)
	}
}
