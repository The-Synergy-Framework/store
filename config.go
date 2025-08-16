package store

import (
	"time"

	"core/metrics"
)

// BaseConfig contains configuration fields common to all store adapters.
type BaseConfig struct {
	// Basic connection info
	Type     string // adapter type (postgres, mysql, sqlite, redis, memory, etc.)
	Host     string
	Port     int
	Username string // some use User, some Username
	Password string

	// Connection pooling (concepts apply to most backends)
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Timeouts
	ConnectTimeout time.Duration

	// Backend-specific options
	Options map[string]string

	// Observability
	EnableMetrics bool
	MetricLabels  metrics.Labels
}

// ConfigOption configures a base config.
type ConfigOption func(*BaseConfig)

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool, labels metrics.Labels) ConfigOption {
	return func(c *BaseConfig) {
		c.EnableMetrics = enabled
		c.MetricLabels = labels
	}
}

// WithPooling configures connection pooling.
func WithPooling(maxIdle int, maxLifetime, maxIdleTime time.Duration) ConfigOption {
	return func(c *BaseConfig) {
		c.MaxIdleConns = maxIdle
		c.ConnMaxLifetime = maxLifetime
		c.ConnMaxIdleTime = maxIdleTime
	}
}

// WithTimeouts configures operation timeouts.
func WithTimeouts(connect time.Duration) ConfigOption {
	return func(c *BaseConfig) {
		c.ConnectTimeout = connect
	}
}

// WithConnection configures basic connection parameters.
func WithConnection(host string, port int, username, password string) ConfigOption {
	return func(c *BaseConfig) {
		c.Host = host
		c.Port = port
		c.Username = username
		c.Password = password
	}
}

// WithOptions sets adapter-specific options.
func WithOptions(options map[string]string) ConfigOption {
	return func(c *BaseConfig) {
		if c.Options == nil {
			c.Options = make(map[string]string)
		}
		for k, v := range options {
			c.Options[k] = v
		}
	}
}

// DefaultConfig returns a base configuration with sensible defaults.
func DefaultConfig() BaseConfig {
	return BaseConfig{
		Host:            "localhost",
		Port:            0, // Backend-specific default
		MaxIdleConns:    10,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnectTimeout:  30 * time.Second,
		Options:         make(map[string]string),
		EnableMetrics:   false,
		MetricLabels:    make(metrics.Labels),
	}
}
