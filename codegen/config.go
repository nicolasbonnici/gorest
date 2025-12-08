package codegen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolasbonnici/gorest/config"
)

func LoadConfig() (*config.Config, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	return config.Load(projectRoot)
}

func GetModelsPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.Models), nil
}

func GetResourcesPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.Resources), nil
}

func GetDTOsPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.DTOs), nil
}

func GetOpenAPIPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.OpenAPI), nil
}

func GetConfigPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.Config), nil
}

func GetRoutesPath(cfg *config.Config) (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, cfg.Codegen.Output.Resources, "routes.go"), nil
}

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
