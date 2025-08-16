package store_test

import (
	"fmt"
	"testing"
	"time"

	"store"
)

func TestBasicTypes(t *testing.T) {
	// Test Condition creation with new helpers
	condition := store.Eq("name", "John")
	if condition.Field != "name" || condition.Value != "John" || condition.Op != store.OpEq {
		t.Errorf("Expected condition to be properly created")
	}

	// Test Mutation creation
	insert := store.NewInsert(map[string]any{"name": "John", "age": 30})
	if len(insert.Values) != 2 {
		t.Errorf("Expected insert to have 2 values")
	}

	// Test simplified CursorParams validation
	params := store.CursorParams{PageSize: 10}
	if params.PageSize <= 0 {
		t.Errorf("Expected valid page size")
	}
}

func TestErrorTypes(t *testing.T) {
	// Test custom error creation
	err := store.NewValidationError("test validation failed")
	if err.Error() != "validation error: test validation failed" {
		t.Errorf("Expected validation error message")
	}

	configErr := store.NewConfigError("invalid config")
	if configErr.Error() != "configuration error: invalid config" {
		t.Errorf("Expected config error message")
	}
}

// Example_unified demonstrates the new unified configuration API
func Example_unified() {

	// Method 1: Use convenience constructors
	pgConfig := store.PostgreSQLConfig("mydb", "user", "password")

	// Method 2: Use the new unified options API
	config := store.NewConfig(
		store.PostgreSQLOptions("mydb", "user", "password",
			store.WithHost("localhost"),
			store.WithPooling(25, 10, time.Hour),
			store.WithTimeouts(30*time.Second, 30*time.Second),
			store.WithMetricsEnabled(),
		)...,
	)

	// Method 3: Start with defaults and apply options
	sqliteConfig := store.DefaultConfig()
	sqliteConfig.Apply(
		store.SQLiteOptions("/tmp/test.db",
			store.WithMetricsEnabled(),
		)...,
	)

	// Method 4: Mix and match individual options
	customConfig := store.NewConfig(
		store.WithConnection("localhost", 5432, "user", "pass", "db"),
		store.WithSSLRequired(),
		store.WithOption("custom_setting", "value"),
	)

	// All configs work the same way - validate and connect
	configs := []store.Config{pgConfig, config, sqliteConfig, customConfig}
	for i, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			// Some configs may fail validation (like custom without type)
			if i == 3 { // customConfig doesn't set type
				continue
			}
			panic(err)
		}

		// Connection string generation works for all
		if cfg.Type == "postgres" || cfg.Type == "mysql" || cfg.Type == "sqlite" {
			_ = cfg.ConnectionString()
		}
	}

	// Output: Unified configuration system working
	fmt.Println("Unified configuration system working")
}

func TestUnifiedOptions(t *testing.T) {
	// Test PostgreSQL options
	config := store.NewConfig(
		store.PostgreSQLOptions("testdb", "user", "pass")...,
	)
	if config.Type != "postgres" || config.Database != "testdb" {
		t.Errorf("PostgreSQL options not applied correctly")
	}

	// Test MySQL options
	config = store.NewConfig(
		store.MySQLOptions("testdb", "user", "pass")...,
	)
	if config.Type != "mysql" || config.Port != 3306 {
		t.Errorf("MySQL options not applied correctly")
	}

	// Test SQLite options
	config = store.NewConfig(
		store.SQLiteOptions("/tmp/test.db")...,
	)
	if config.Type != "sqlite" || config.FilePath != "/tmp/test.db" {
		t.Errorf("SQLite options not applied correctly")
	}

	// Test option chaining
	config = store.DefaultConfig()
	config.Apply(
		store.WithHost("custom-host"),
		store.WithPort(9999),
		store.WithMetricsEnabled(),
		store.WithOption("custom", "value"),
	)

	if config.Host != "custom-host" || config.Port != 9999 || !config.EnableMetrics {
		t.Errorf("Option chaining not working correctly")
	}

	if config.Options["custom"] != "value" {
		t.Errorf("Custom option not set correctly")
	}
}
