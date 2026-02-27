package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	// Server
	Listen    string `json:"listen"`
	JWTSecret string `json:"jwt_secret"`
	DBPath    string `json:"db_path"`

	// Admin
	AdminUser string `json:"admin_user"`
	AdminPass string `json:"admin_pass"`

	// Game defaults
	GameServerURL string `json:"game_server_url"`
	ClientVersion string `json:"client_version"`

	// Paths
	DataDir       string `json:"-"`
	GameConfigDir string `json:"-"`
}

func DefaultConfig() *Config {
	return &Config{
		Listen:        "0.0.0.0:8080",
		JWTSecret:     "qq-farm-bot-secret-change-me",
		DBPath:        "data/farm.db",
		AdminUser:     "admin",
		AdminPass:     "admin123",
		GameServerURL: "wss://gate-obt.nqf.qq.com/prod/ws",
		ClientVersion: "1.6.0.14_20251224",
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) ResolvePaths(baseDir string) {
	c.DataDir = filepath.Join(baseDir, "data")
	c.GameConfigDir = filepath.Join(baseDir, "gameConfig")
	if !filepath.IsAbs(c.DBPath) {
		c.DBPath = filepath.Join(baseDir, c.DBPath)
	}
	os.MkdirAll(c.DataDir, 0755)
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}
