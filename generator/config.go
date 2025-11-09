package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type OutputPaths struct {
	Models    string `yaml:"models"`
	Resources string `yaml:"resources"`
	DTOs      string `yaml:"dtos"`
	OpenAPI   string `yaml:"openapi"`
	Config    string `yaml:"config"`
}

type Config struct {
	Output OutputPaths `yaml:"output"`
}

var defaultConfig = Config{
	Output: OutputPaths{
		Models:    "models",
		Resources: "resources",
		DTOs:      "dtos",
		OpenAPI:   "openapi",
		Config:    "config",
	},
}

func LoadConfig() (*Config, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	configPath := filepath.Join(projectRoot, ".gorest.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Output.Models == "" {
		cfg.Output.Models = defaultConfig.Output.Models
	}
	if cfg.Output.Resources == "" {
		cfg.Output.Resources = defaultConfig.Output.Resources
	}
	if cfg.Output.DTOs == "" {
		cfg.Output.DTOs = defaultConfig.Output.DTOs
	}
	if cfg.Output.OpenAPI == "" {
		cfg.Output.OpenAPI = defaultConfig.Output.OpenAPI
	}
	if cfg.Output.Config == "" {
		cfg.Output.Config = defaultConfig.Output.Config
	}

	return &cfg, nil
}

func (c *Config) GetModelsPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.Models), nil
}

func (c *Config) GetResourcesPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.Resources), nil
}

func (c *Config) GetDTOsPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.DTOs), nil
}

func (c *Config) GetOpenAPIPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.OpenAPI), nil
}

func (c *Config) GetConfigPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.Config), nil
}

func (c *Config) GetRoutesPath() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, c.Output.Resources, "routes.go"), nil
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
