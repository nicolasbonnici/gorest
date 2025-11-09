package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
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
	counts := []int{10, 100, 1000}
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

	// Discard server output (we don't need it unless debugging)
	serverCmd.Stdout = nil
	serverCmd.Stderr = nil

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

		// Use Vegeta for health check
		target := vegeta.Target{
			Method: "GET",
			URL:    "http://localhost:3001/health",
		}
		attacker := vegeta.NewAttacker()

		for res := range attacker.Attack(vegeta.NewStaticTargeter(target), vegeta.Rate{Freq: 1, Per: time.Second}, 1*time.Second, "Health Check") {
			if res.Code == 200 {
				serverReady = true
				break
			}
		}

		if serverReady {
			break
		}
	}

	if !serverReady {
		fmt.Println("ERROR: Server failed to start within 15 seconds")
		os.Exit(1)
	}
	fmt.Println("[INFO] Server is ready")

	// Give server extra time to fully initialize all routes
	time.Sleep(2 * time.Second)

	// Verify the endpoint exists with a test request
	target := vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:3001/benchmarkitems?limit=1",
	}
	attacker := vegeta.NewAttacker()
	for res := range attacker.Attack(vegeta.NewStaticTargeter(target), vegeta.Rate{Freq: 1, Per: time.Second}, 1*time.Second, "Endpoint Check") {
		if res.Code != 200 {
			fmt.Printf("WARNING: Endpoint check failed with status %d\n", res.Code)
			time.Sleep(2 * time.Second)
		}
		break
	}

	// Run benchmarks
	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("       Running Benchmarks")
	fmt.Println("=========================================")
	fmt.Println()

	// Test different limits with multiple concurrency levels
	concurrencyLevels := []int{1, 10, 50}
	testDuration := 5 * time.Second

	for _, limit := range counts {
		fmt.Printf("Benchmarking GET /benchmarkitems?limit=%d\n", limit)
		fmt.Println("─────────────────────────────────────────────────────────────────")

		for _, concurrency := range concurrencyLevels {
			url := fmt.Sprintf("http://localhost:3001/benchmarkitems?limit=%d", limit)

			// Create target
			target := vegeta.Target{
				Method: "GET",
				URL:    url,
			}
			targeter := vegeta.NewStaticTargeter(target)

			// Configure attack rate
			rate := vegeta.Rate{Freq: concurrency, Per: time.Second}

			// Create attacker
			attacker := vegeta.NewAttacker()

			// Run attack and collect metrics
			var metrics vegeta.Metrics
			for res := range attacker.Attack(targeter, rate, testDuration, fmt.Sprintf("Load Test (concurrency=%d)", concurrency)) {
				metrics.Add(res)
			}
			metrics.Close()

			// Display results
			fmt.Printf("  Concurrency: %-3d | ", concurrency)
			fmt.Printf("RPS: %7.0f | ", metrics.Rate)
			fmt.Printf("p50: %8s | ", metrics.Latencies.P50)
			fmt.Printf("p95: %8s | ", metrics.Latencies.P95)
			fmt.Printf("p99: %8s | ", metrics.Latencies.P99)

			if len(metrics.Errors) > 0 {
				errorRate := float64(len(metrics.Errors)) / float64(metrics.Requests) * 100
				fmt.Printf("Errors: %.2f%% ", errorRate)
				// Show first error for debugging
				for _, err := range metrics.Errors {
					fmt.Printf("(%s)", err)
					break
				}
			} else {
				fmt.Printf("Errors: 0")
			}
			fmt.Printf(" | Total: %d\n", metrics.Requests)
		}
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
