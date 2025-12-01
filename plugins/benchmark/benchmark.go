package benchmark

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/generator"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/pluginloader"
)

func init() {
	pluginloader.RegisterPluginFactory("benchmark", NewPlugin)
}

// BenchmarkPlugin provides API performance benchmarking
type BenchmarkPlugin struct {
	db database.Database
}

func NewPlugin() plugin.Plugin {
	return &BenchmarkPlugin{}
}

func (p *BenchmarkPlugin) Name() string {
	return "benchmark"
}

func (p *BenchmarkPlugin) Initialize(cfg map[string]interface{}) error {
	if db, ok := cfg["database"].(database.Database); ok {
		p.db = db
	}
	return nil
}

// Handler returns a no-op middleware since benchmark is a CLI tool
func (p *BenchmarkPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// Commands implements the CommandProvider interface
func (p *BenchmarkPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		&BenchmarkCommand{plugin: p},
	}
}

// BenchmarkCommand runs API performance benchmarks
type BenchmarkCommand struct {
	plugin *BenchmarkPlugin
}

func (c *BenchmarkCommand) Name() string {
	return "benchmark"
}

func (c *BenchmarkCommand) Description() string {
	return "Run API performance benchmarks with different load levels"
}

func (c *BenchmarkCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	db := c.plugin.db
	if db == nil {
		return &plugin.CommandResult{
			Success: false,
			Error:   fmt.Errorf("database not configured"),
			Message: "Database connection required for benchmarks",
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return &plugin.CommandResult{
			Success: false,
			Error:   fmt.Errorf("DATABASE_URL not set"),
			Message: "DATABASE_URL environment variable required",
		}
	}

	dbCtx := context.Background()

	fmt.Println("=========================================")
	fmt.Println("      API Performance Benchmark")
	fmt.Println("=========================================")
	fmt.Println()

	// Setup benchmark table
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Setting up benchmark table...")
	}

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
		return &plugin.CommandResult{
			Success: false,
			Error:   err,
			Message: "Failed to create benchmark table",
		}
	}

	// Generate test data
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Generating test data...")
	}

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
			return &plugin.CommandResult{
				Success: false,
				Error:   err,
				Message: "Failed to insert test data",
			}
		}
	}

	// Generate models and resources using generator library
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Generating models and resources...")
	}

	tables := generator.LoadSchema(c.plugin.db)
	generator.GenerateStructs(tables)

	authCfg := generator.DefaultAuthConfig()
	generator.GenerateAPI(authCfg)

	// Build and start API server
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Building API server...")
	}

	cmd := exec.Command("go", "build", "-o", "./bin/benchmark-server", "./cmd/benchmark/testserver/main.go")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return &plugin.CommandResult{
			Success: false,
			Error:   err,
			Message: "Failed to build benchmark server",
		}
	}

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Starting API server...")
	}

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
		return &plugin.CommandResult{
			Success: false,
			Error:   err,
			Message: "Failed to start server",
		}
	}
	defer func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}()

	// Wait for server to be ready
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Waiting for server to be ready...")
	}

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
		return &plugin.CommandResult{
			Success: false,
			Error:   fmt.Errorf("server startup timeout"),
			Message: "Server failed to start within 15 seconds",
		}
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
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Cleaning up...")
	}

	db.Exec(dbCtx, "DROP TABLE IF EXISTS benchmark_items CASCADE")

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Restoring original schema...")
	}

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

	return &plugin.CommandResult{
		Success: true,
		Message: "Benchmark completed successfully",
	}
}
