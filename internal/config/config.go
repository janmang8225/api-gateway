package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Path     string   `yaml:"path"`
	Auth     bool     `yaml:"auth"`
	Backends []string `yaml:"backends"`
}

type Config struct {
	Port      int     `yaml:"port"`
	JWTSecret string  `yaml:"jwt_secret"`
	Routes    []Route `yaml:"routes"`
}

type Manager struct {
	mu       sync.RWMutex
	current  *Config
	filePath string
}

func NewManager(filePath string) (*Manager, error) {
	cfg, err := Load(filePath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		current:  cfg,
		filePath: filePath,
	}, nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) Reload() error {
	cfg, err := Load(m.filePath)
	if err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}

	m.mu.Lock()
	m.current = cfg
	m.mu.Unlock()

	fmt.Println("config: reloaded successfully")
	return nil
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &cfg, nil
}