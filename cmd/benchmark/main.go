package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/nicolasbonnici/gorest/pkg/database"
	_ "github.com/nicolasbonnici/gorest/pkg/database/mysql"
	_ "github.com/nicolasbonnici/gorest/pkg/database/postgres"
	_ "github.com/nicolasbonnici/gorest/pkg/database/sqlite"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("ERROR: DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	db, err := database.Open("", dbURL)
	if err != nil {
		fmt.Printf("ERROR: Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	fmt.Println("=========================================")
	fmt.Println("      API Performance Benchmark")
	fmt.Println("=========================================")
	fmt.Println()

	// Create benchmark table
	fmt.Println("[INFO] Setting up benchmark table...")
	db.Exec(ctx, "DROP TABLE IF EXISTS benchmark_items CASCADE")
	_, err = db.Exec(ctx, `
		CREATE TABLE benchmark_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT,
			value INTEGER,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		fmt.Printf("ERROR: Failed to create benchmark table: %v\n", err)
		os.Exit(1)
	}

	// Generate test data
	fmt.Println("[INFO] Generating test data...")
	counts := []int{10, 100, 1000, 10000}
	maxCount := counts[len(counts)-1]

	for i := 1; i <= maxCount; i++ {
		_, err := db.Exec(ctx,
			"INSERT INTO benchmark_items (name, value, description) VALUES ($1, $2, $3)",
			fmt.Sprintf("Item %d", i),
			i,
			fmt.Sprintf("Description for item %d with some additional text to make it realistic", i),
		)
		if err != nil {
			fmt.Printf("ERROR: Failed to insert data: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate models and resources
	fmt.Println("[INFO] Generating models and resources...")
	cmd := exec.Command("go", "run", "./cmd/modelgen/main.go")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		fmt.Printf("ERROR: modelgen failed: %v\n", err)
		os.Exit(1)
	}

	cmd = exec.Command("go", "run", "./cmd/resourcegen/main.go", "-y")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		fmt.Printf("ERROR: resourcegen failed: %v\n", err)
		os.Exit(1)
	}

	// Build and start API server
	fmt.Println("[INFO] Building API server...")
	cmd = exec.Command("make", "build")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		fmt.Printf("ERROR: build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Starting API server...")
	serverCmd := exec.Command("./bin/gorest")
	serverCmd.Env = append(os.Environ(),
		"DATABASE_URL="+dbURL,
		"PORT=3001",
		"JWT_SECRET=bmk-jwt-k3y-f0r-4p1-p3rf0rm4nc3-m34sur3m3nts-0nly",
		"JWT_TTL=3600",
		"PAGINATION_LIMIT=50",
		"PAGINATION_MAX_LIMIT=10000",
		"CORS_ORIGINS=*",
		"ENVIRONMENT=test",
	)

	// Capture output for debugging
	logFile, _ := os.Create("/tmp/gorest-benchmark-server.log")
	serverCmd.Stdout = logFile
	serverCmd.Stderr = logFile

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("ERROR: Failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}()

	// Wait for server to be ready with health check polling
	fmt.Println("[INFO] Waiting for server to be ready...")
	serverReady := false
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://localhost:3001/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				serverReady = true
				break
			}
		}
	}

	if !serverReady {
		fmt.Println("ERROR: Server failed to start within 15 seconds")
		fmt.Println("Server logs:")
		logContent, _ := os.ReadFile("/tmp/gorest-benchmark-server.log")
		fmt.Println(string(logContent))
		os.Exit(1)
	}
	fmt.Println("[INFO] Server is ready")
	logFile.Close()

	// Run benchmarks
	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("       Running Benchmarks")
	fmt.Println("=========================================")
	fmt.Println()

	for _, limit := range counts {
		fmt.Printf("Benchmarking GET /api/benchmark_items?limit=%d\n", limit)

		url := fmt.Sprintf("http://localhost:3001/api/benchmark_items?limit=%d", limit)

		// Warm-up request
		http.Get(url)

		// Measure 5 requests and average
		var totalDuration time.Duration
		for i := 0; i < 5; i++ {
			start := time.Now()
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("  ERROR: Request failed: %v\n", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				fmt.Printf("  ERROR: Failed to read response: %v\n", err)
				continue
			}

			duration := time.Since(start)
			totalDuration += duration

			// Parse to verify response
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Printf("  ERROR: Failed to parse response: %v\n", err)
				continue
			}
		}

		avgDuration := totalDuration / 5
		fmt.Printf("  Average response time: %v\n", avgDuration)
		fmt.Printf("  Throughput: %.2f req/s\n", 1.0/avgDuration.Seconds())
		fmt.Println()
	}

	fmt.Println("=========================================")
	fmt.Println("       Benchmark Complete")
	fmt.Println("=========================================")

	fmt.Println("\n[INFO] Cleaning up...")
	db.Exec(ctx, "DROP TABLE IF EXISTS benchmark_items CASCADE")

	fmt.Println("[INFO] Restoring original schema...")
	cmd = exec.Command("make", "test-schema")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Run()

	cmd = exec.Command("make", "test-generate")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Run()

	fmt.Println("[INFO] Cleanup complete")
}
