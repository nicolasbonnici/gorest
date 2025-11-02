package main

import (
	"context"
	"fmt"
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
	fmt.Println("   API Resource Generation Benchmark")
	fmt.Println("=========================================")
	fmt.Println()

	benchmarks := []int{1, 10, 100, 1000}

	for _, count := range benchmarks {
		fmt.Printf("Benchmarking %d resources...\n", count)

		for i := 1; i <= count; i++ {
			tableName := fmt.Sprintf("bench_%d", i)
			db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
			_, err := db.Exec(ctx, fmt.Sprintf(
				"CREATE TABLE %s (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT, value INTEGER, created_at TIMESTAMP DEFAULT NOW())",
				tableName,
			))
			if err != nil {
				fmt.Printf("ERROR: Failed to create table %s: %v\n", tableName, err)
				os.Exit(1)
			}
		}

		modelgenStart := time.Now()
		cmd := exec.Command("go", "run", "./cmd/modelgen/main.go")
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			fmt.Printf("ERROR: modelgen failed: %v\n", err)
			os.Exit(1)
		}
		modelgenDuration := time.Since(modelgenStart)

		resourcegenStart := time.Now()
		cmd = exec.Command("go", "run", "./cmd/resourcegen/main.go", "-y")
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			fmt.Printf("ERROR: resourcegen failed: %v\n", err)
			os.Exit(1)
		}
		resourcegenDuration := time.Since(resourcegenStart)

		for i := 1; i <= count; i++ {
			tableName := fmt.Sprintf("bench_%d", i)
			db.Exec(ctx, fmt.Sprintf("DROP TABLE %s", tableName))
		}

		fmt.Printf("  modelgen:    %v\n", modelgenDuration)
		fmt.Printf("  resourcegen: %v\n", resourcegenDuration)
		fmt.Printf("  total:       %v\n", modelgenDuration+resourcegenDuration)
		fmt.Println()
	}

	fmt.Println("=========================================")
	fmt.Println("         Benchmark Complete")
	fmt.Println("=========================================")

	fmt.Println("\n[INFO] Restoring original schema...")
	cmd := exec.Command("make", "test-schema")
	cmd.Run()
	cmd = exec.Command("make", "test-generate")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Run()
	fmt.Println("[INFO] Cleanup complete")
}
