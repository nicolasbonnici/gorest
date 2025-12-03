package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/generator"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.Open("", cfg.Database.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	commandName := os.Args[1]

	// Route to command handler
	switch commandName {
	case "models":
		if err := modelsCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "resources":
		if err := resourcesCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "openapi":
		if err := openapiCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "all":
		if err := allCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "benchmark":
		if err := benchmarkCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", commandName)
		printUsage()
		os.Exit(1)
	}
}

func modelsCommand(db database.Database) error {
	progress("Loading database schema...")
	tables := generator.LoadSchema(db)

	progress("Generating model structs...")
	generator.GenerateStructs(tables)

	progress("Models generated successfully")
	fmt.Println("Model generation completed successfully")
	return nil
}

func resourcesCommand(db database.Database) error {
	progress("Generating API resources...")
	authCfg := generator.DefaultAuthConfig()
	generator.GenerateAPI(authCfg)

	progress("Resources generated successfully")
	fmt.Println("Resource generation completed successfully")
	return nil
}

func openapiCommand(db database.Database) error {
	progress("Generating OpenAPI schema...")
	tables := generator.LoadSchema(db)
	generator.GenerateOpenAPI(tables)

	progress("OpenAPI schema generated successfully")
	fmt.Println("OpenAPI generation completed successfully")
	return nil
}

func allCommand(db database.Database) error {
	progress("Running: models")
	if err := modelsCommand(db); err != nil {
		return err
	}

	progress("Running: resources")
	if err := resourcesCommand(db); err != nil {
		return err
	}

	progress("Running: openapi")
	if err := openapiCommand(db); err != nil {
		return err
	}

	fmt.Println("All code generation completed successfully")
	return nil
}

func benchmarkCommand(db database.Database) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable required")
	}

	dbCtx := context.Background()

	fmt.Println("=========================================")
	fmt.Println("      API Performance Benchmark")
	fmt.Println("=========================================")
	fmt.Println()

	// Setup benchmark table
	progress("Setting up benchmark table...")
	db.Exec(dbCtx, "DROP TABLE IF EXISTS benchmark_items CASCADE")
	_, err := db.Exec(dbCtx, `
		CREATE TABLE benchmark_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT,
			value INTEGER,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create benchmark table: %w", err)
	}

	// Generate test data
	progress("Generating test data...")
	counts := []int{10, 100, 1000}
	maxCount := counts[len(counts)-1]

	for i := 1; i <= maxCount; i++ {
		_, err := db.Exec(dbCtx,
			"INSERT INTO benchmark_items (name, value, description) VALUES ($1, $2, $3)",
			fmt.Sprintf("Item %d", i),
			i,
			fmt.Sprintf("Description for item %d with some additional text to make it realistic", i),
		)
		if err != nil {
			return fmt.Errorf("failed to insert test data: %w", err)
		}
	}

	// Generate models and resources
	progress("Generating models and resources...")
	tables := generator.LoadSchema(db)
	generator.GenerateStructs(tables)

	authCfg := generator.DefaultAuthConfig()
	generator.GenerateAPI(authCfg)

	// Build and start API server
	progress("Building API server...")
	cmd := exec.Command("go", "build", "-o", "./bin/benchmark-server", "./cmd/benchmark/testserver/main.go")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build benchmark server: %w", err)
	}

	progress("Starting API server...")
	serverCmd := exec.Command("./bin/benchmark-server")
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
	serverCmd.Stdout = nil
	serverCmd.Stderr = nil

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}()

	// Wait for server to be ready
	progress("Waiting for server to be ready...")
	serverReady := false
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)

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
		return fmt.Errorf("server startup timeout")
	}

	time.Sleep(2 * time.Second)

	// Verify endpoint
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

	concurrencyLevels := []int{1, 10, 50}
	testDuration := 5 * time.Second

	for _, limit := range counts {
		fmt.Printf("Benchmarking GET /benchmarkitems?limit=%d\n", limit)
		fmt.Println("─────────────────────────────────────────────────────────────────")

		for _, concurrency := range concurrencyLevels {
			url := fmt.Sprintf("http://localhost:3001/benchmarkitems?limit=%d", limit)

			target := vegeta.Target{
				Method: "GET",
				URL:    url,
			}
			targeter := vegeta.NewStaticTargeter(target)
			rate := vegeta.Rate{Freq: concurrency, Per: time.Second}
			attacker := vegeta.NewAttacker()

			var metrics vegeta.Metrics
			for res := range attacker.Attack(targeter, rate, testDuration, fmt.Sprintf("Load Test (concurrency=%d)", concurrency)) {
				metrics.Add(res)
			}
			metrics.Close()

			fmt.Printf("  Concurrency: %-3d | ", concurrency)
			fmt.Printf("RPS: %7.0f | ", metrics.Rate)
			fmt.Printf("p50: %8s | ", metrics.Latencies.P50)
			fmt.Printf("p95: %8s | ", metrics.Latencies.P95)
			fmt.Printf("p99: %8s | ", metrics.Latencies.P99)

			if len(metrics.Errors) > 0 {
				errorRate := float64(len(metrics.Errors)) / float64(metrics.Requests) * 100
				fmt.Printf("Errors: %.2f%% ", errorRate)
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

	// Cleanup
	progress("Cleaning up...")
	db.Exec(dbCtx, "DROP TABLE IF EXISTS benchmark_items CASCADE")

	progress("Restoring original schema...")
	cmd = exec.Command("make", "test-schema")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Run()

	cmd = exec.Command("make", "test-generate")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Run()

	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("       Benchmark Complete")
	fmt.Println("=========================================")

	fmt.Println("Benchmark completed successfully")
	return nil
}

func progress(message string) {
	fmt.Printf("  → %s\n", message)
}

func printUsage() {
	fmt.Println("GoREST Code Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gorest-codegen <command>")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  models      Generate model structs from database schema")
	fmt.Println("  resources   Generate REST API resources and DTOs from models")
	fmt.Println("  openapi     Generate OpenAPI schema file")
	fmt.Println("  all         Run all code generation steps")
	fmt.Println("  benchmark   Run API performance benchmarks")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gorest-codegen models")
	fmt.Println("  gorest-codegen resources")
	fmt.Println("  gorest-codegen all")
	fmt.Println("  gorest-codegen benchmark")
}
