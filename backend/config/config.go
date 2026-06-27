package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

// PostgresConf - config for the Postgres server
type PostgresConf struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// RedisConf - config for the Redis server
type RedisConf struct {
	Host      string `json:"host"`
	Pass      string `json:"pass"`
	DB        int    `json:"logical_database"`
	CharFloor int    `json:"char_floor"`
}

// AuthConf - authentication configuration
type AuthConf struct {
	JWTSecret            string `json:"jwt_secret"`
	AdminEmail           string `json:"admin_email"`
	AdminPassword        string `json:"admin_password"`
	TokenExpiryHrs       int    `json:"token_expiry_hours"`
	DisablePasswordLogin bool   `json:"disable_password_login"`
}

// GatewayConf - REST API gateway configuration
type GatewayConf struct {
	HTTPAddr string `json:"http_addr"`
}

// Configuration - the server config
type Configuration struct {
	ListenInterface string       `json:"listen_interface"`
	Redis           RedisConf    `json:"redis_conf"`
	Postgres        PostgresConf `json:"postgres_conf"`
	Auth            AuthConf     `json:"auth_conf"`
	Gateway         GatewayConf  `json:"gateway_conf"`
	GRPCHost        string       `json:"grpc_host"`
}

// DSN returns a Postgres connection string.
func (p PostgresConf) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, p.SSLMode)
}

// GetJWTSecret returns the JWT secret from config or env var override.
func (a AuthConf) GetJWTSecret() string {
	if env := os.Getenv("GOSHORTEN_JWT_SECRET"); env != "" {
		return env
	}
	return a.JWTSecret
}

// GetAdminEmail returns the admin email from config or env var override.
func (a AuthConf) GetAdminEmail() string {
	if env := os.Getenv("GOSHORTEN_ADMIN_EMAIL"); env != "" {
		return env
	}
	return a.AdminEmail
}

// GetAdminPassword returns the admin password from config or env var override.
func (a AuthConf) GetAdminPassword() string {
	if env := os.Getenv("GOSHORTEN_ADMIN_PASSWORD"); env != "" {
		return env
	}
	return a.AdminPassword
}

// GetDisablePasswordLogin returns true if password login should be disabled.
// GOSHORTEN_DISABLE_PASSWORD_LOGIN=true overrides the JSON config value.
func (a AuthConf) GetDisablePasswordLogin() bool {
	if v := os.Getenv("GOSHORTEN_DISABLE_PASSWORD_LOGIN"); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return a.DisablePasswordLogin
}

// GetTokenExpiry returns the token expiry in hours, defaulting to 24.
func (a AuthConf) GetTokenExpiry() int {
	if a.TokenExpiryHrs <= 0 {
		return 24
	}
	return a.TokenExpiryHrs
}

// ConfigFromFile parses the given file and returns the config, then applies
// environment variable overrides for container-friendly deployment.
func ConfigFromFile(fileName string) (Configuration, error) {
	var conf Configuration

	confjson, err := os.ReadFile(fileName)
	if err != nil {
		return conf, errors.Wrapf(err, "Failed to open the config file at: %s", fileName)
	}

	if err := json.Unmarshal(confjson, &conf); err != nil {
		return conf, errors.Wrapf(err, "Unable to parse the config file at: %s", fileName)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&conf)

	return conf, nil
}

// applyEnvOverrides lets container orchestrators (Docker, Podman) inject config
// via env vars without modifying config.json.
func applyEnvOverrides(conf *Configuration) {
	if v := os.Getenv("GOSHORTEN_REDIS_HOST"); v != "" {
		conf.Redis.Host = v
	}
	if v := os.Getenv("GOSHORTEN_REDIS_PASS"); v != "" {
		conf.Redis.Pass = v
	}
	if v := os.Getenv("GOSHORTEN_POSTGRES_HOST"); v != "" {
		conf.Postgres.Host = v
	}
	if v := os.Getenv("GOSHORTEN_POSTGRES_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			conf.Postgres.Port = port
		}
	}
	if v := os.Getenv("GOSHORTEN_POSTGRES_USER"); v != "" {
		conf.Postgres.User = v
	}
	if v := os.Getenv("GOSHORTEN_POSTGRES_PASSWORD"); v != "" {
		conf.Postgres.Password = v
	}
	if v := os.Getenv("GOSHORTEN_POSTGRES_DB"); v != "" {
		conf.Postgres.DBName = v
	}
	if v := os.Getenv("GOSHORTEN_GRPC_HOST"); v != "" {
		conf.GRPCHost = v
	}
	if v := os.Getenv("GOSHORTEN_GATEWAY_ADDR"); v != "" {
		conf.Gateway.HTTPAddr = v
	}
}
