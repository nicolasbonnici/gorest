package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolasbonnici/gorest/config"
)

// LoadConfig loads the GoREST configuration
func LoadConfig() (*config.Config, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	return config.Load(projectRoot)
}

// GetModelsPath returns the absolute path to the models directory
func GetModelsPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.Models), nil
}

// GetResourcesPath returns the absolute path to the resources directory
func GetResourcesPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.Resources), nil
}

// GetDTOsPath returns the absolute path to the DTOs directory
func GetDTOsPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.DTOs), nil
}

// GetOpenAPIPath returns the absolute path to the OpenAPI directory
func GetOpenAPIPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.OpenAPI), nil
}

// GetConfigPath returns the absolute path to the config directory
func GetConfigPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.Config), nil
}

// GetRoutesPath returns the absolute path to the routes.go file
func GetRoutesPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Generate.Output.Resources, "routes.go"), nil
}

// getModuleName reads the module name from go.mod file
func getModuleName() string {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "github.com/nicolasbonnici/gorest"
	}

	goModPath := filepath.Join(projectRoot, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return "github.com/nicolasbonnici/gorest"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return "github.com/nicolasbonnici/gorest"
}
