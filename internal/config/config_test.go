package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalEnvVars := make(map[string]string)
	envVarsToTest := []string{
		"SERVER_HOST", "SERVER_PORT", "DATABASE_HOST", "DATABASE_PORT",
		"DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_DBNAME", "DATABASE_SSLMODE",
		"NATS_URL", "LOG_LEVEL", "LOG_JSON",
	}

	for _, envVar := range envVarsToTest {
		originalEnvVars[envVar] = os.Getenv(envVar)
	}

	// Очищаем переменные окружения для чистого теста
	defer func() {
		for envVar, originalValue := range originalEnvVars {
			if originalValue == "" {
				os.Unsetenv(envVar)
			} else {
				os.Setenv(envVar, originalValue)
			}
		}
	}()

	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig *Config
		expectedError  bool
	}{
		{
			name:    "default_values",
			envVars: map[string]string{},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
				NATS: NATSConfig{
					URL: "nats://localhost:4222",
				},
				Log: LogConfig{
					Level: "info",
					JSON:  false,
				},
			},
			expectedError: false,
		},
		{
			name: "custom_server_config",
			envVars: map[string]string{
				"SERVER_HOST": "127.0.0.1",
				"SERVER_PORT": "9090",
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "127.0.0.1",
					Port: 9090,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
				NATS: NATSConfig{
					URL: "nats://localhost:4222",
				},
				Log: LogConfig{
					Level: "info",
					JSON:  false,
				},
			},
			expectedError: false,
		},
		{
			name: "custom_database_config",
			envVars: map[string]string{
				"DATABASE_HOST":     "db.example.com",
				"DATABASE_PORT":     "5433",
				"DATABASE_USER":     "testuser",
				"DATABASE_PASSWORD": "testpass",
				"DATABASE_DBNAME":   "testdb",
				"DATABASE_SSLMODE":  "require",
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "db.example.com",
					Port:     5433,
					User:     "testuser",
					Password: "testpass",
					DBName:   "testdb",
					SSLMode:  "require",
				},
				NATS: NATSConfig{
					URL: "nats://localhost:4222",
				},
				Log: LogConfig{
					Level: "info",
					JSON:  false,
				},
			},
			expectedError: false,
		},
		{
			name: "custom_nats_config",
			envVars: map[string]string{
				"NATS_URL": "nats://nats.example.com:4222",
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
				NATS: NATSConfig{
					URL: "nats://nats.example.com:4222",
				},
				Log: LogConfig{
					Level: "info",
					JSON:  false,
				},
			},
			expectedError: false,
		},
		{
			name: "custom_log_config",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
				"LOG_JSON":  "true",
			},
			expectedConfig: &Config{
				Server: ServerConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
				NATS: NATSConfig{
					URL: "nats://localhost:4222",
				},
				Log: LogConfig{
					Level: "debug",
					JSON:  true,
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем все переменные окружения
			for _, envVar := range envVarsToTest {
				os.Unsetenv(envVar)
			}

			// Устанавливаем переменные окружения для теста
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config, err := Load()

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("expected config, but got nil")
				return
			}

			// Проверяем Server конфигурацию
			if config.Server.Host != tt.expectedConfig.Server.Host {
				t.Errorf("expected server host '%s', but got '%s'", tt.expectedConfig.Server.Host, config.Server.Host)
			}
			if config.Server.Port != tt.expectedConfig.Server.Port {
				t.Errorf("expected server port %d, but got %d", tt.expectedConfig.Server.Port, config.Server.Port)
			}

			// Проверяем Database конфигурацию
			if config.Database.Host != tt.expectedConfig.Database.Host {
				t.Errorf("expected database host '%s', but got '%s'", tt.expectedConfig.Database.Host, config.Database.Host)
			}
			if config.Database.Port != tt.expectedConfig.Database.Port {
				t.Errorf("expected database port %d, but got %d", tt.expectedConfig.Database.Port, config.Database.Port)
			}
			if config.Database.User != tt.expectedConfig.Database.User {
				t.Errorf("expected database user '%s', but got '%s'", tt.expectedConfig.Database.User, config.Database.User)
			}
			if config.Database.Password != tt.expectedConfig.Database.Password {
				t.Errorf("expected database password '%s', but got '%s'", tt.expectedConfig.Database.Password, config.Database.Password)
			}
			if config.Database.DBName != tt.expectedConfig.Database.DBName {
				t.Errorf("expected database name '%s', but got '%s'", tt.expectedConfig.Database.DBName, config.Database.DBName)
			}
			if config.Database.SSLMode != tt.expectedConfig.Database.SSLMode {
				t.Errorf("expected database ssl mode '%s', but got '%s'", tt.expectedConfig.Database.SSLMode, config.Database.SSLMode)
			}

			// Проверяем NATS конфигурацию
			if config.NATS.URL != tt.expectedConfig.NATS.URL {
				t.Errorf("expected NATS URL '%s', but got '%s'", tt.expectedConfig.NATS.URL, config.NATS.URL)
			}

			// Проверяем Log конфигурацию
			if config.Log.Level != tt.expectedConfig.Log.Level {
				t.Errorf("expected log level '%s', but got '%s'", tt.expectedConfig.Log.Level, config.Log.Level)
			}
			if config.Log.JSON != tt.expectedConfig.Log.JSON {
				t.Errorf("expected log JSON %t, but got %t", tt.expectedConfig.Log.JSON, config.Log.JSON)
			}
		})
	}
}

func TestDatabaseDSN(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedDSN string
	}{
		{
			name: "default_config",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
			},
			expectedDSN: "host=localhost port=5432 user=postgres password=postgres dbname=scoring sslmode=disable",
		},
		{
			name: "custom_config",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "db.example.com",
					Port:     5433,
					User:     "testuser",
					Password: "testpass",
					DBName:   "testdb",
					SSLMode:  "require",
				},
			},
			expectedDSN: "host=db.example.com port=5433 user=testuser password=testpass dbname=testdb sslmode=require",
		},
		{
			name: "special_characters_in_password",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "user@domain",
					Password: "pass@word#123",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
			},
			expectedDSN: "host=localhost port=5432 user=user@domain password=pass@word#123 dbname=scoring sslmode=disable",
		},
		{
			name: "empty_password",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "",
					DBName:   "scoring",
					SSLMode:  "disable",
				},
			},
			expectedDSN: "host=localhost port=5432 user=postgres password= dbname=scoring sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DatabaseDSN()
			if dsn != tt.expectedDSN {
				t.Errorf("expected DSN '%s', but got '%s'", tt.expectedDSN, dsn)
			}
		})
	}
}

func TestInvalidPortConfiguration(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalServerPort := os.Getenv("SERVER_PORT")
	originalDatabasePort := os.Getenv("DATABASE_PORT")

	defer func() {
		if originalServerPort == "" {
			os.Unsetenv("SERVER_PORT")
		} else {
			os.Setenv("SERVER_PORT", originalServerPort)
		}
		if originalDatabasePort == "" {
			os.Unsetenv("DATABASE_PORT")
		} else {
			os.Setenv("DATABASE_PORT", originalDatabasePort)
		}
	}()

	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name: "invalid_server_port",
			envVars: map[string]string{
				"SERVER_PORT": "invalid",
			},
		},
		{
			name: "invalid_database_port",
			envVars: map[string]string{
				"DATABASE_PORT": "not_a_number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем переменные окружения
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("DATABASE_PORT")

			// Устанавливаем переменные окружения для теста
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			_, err := Load()

			// Ожидаем ошибку при невалидных портах
			if err == nil {
				t.Error("expected error for invalid port configuration, but got nil")
			}
		})
	}
}

func TestBooleanConfiguration(t *testing.T) {
	// Сохраняем оригинальную переменную окружения
	originalLogJSON := os.Getenv("LOG_JSON")

	defer func() {
		if originalLogJSON == "" {
			os.Unsetenv("LOG_JSON")
		} else {
			os.Setenv("LOG_JSON", originalLogJSON)
		}
	}()

	tests := []struct {
		name         string
		logJSONValue string
		expectedJSON bool
	}{
		{
			name:         "true_value",
			logJSONValue: "true",
			expectedJSON: true,
		},
		{
			name:         "false_value",
			logJSONValue: "false",
			expectedJSON: false,
		},
		{
			name:         "1_value",
			logJSONValue: "1",
			expectedJSON: true,
		},
		{
			name:         "0_value",
			logJSONValue: "0",
			expectedJSON: false,
		},
		{
			name:         "empty_value",
			logJSONValue: "",
			expectedJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logJSONValue == "" {
				os.Unsetenv("LOG_JSON")
			} else {
				os.Setenv("LOG_JSON", tt.logJSONValue)
			}

			config, err := Load()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config.Log.JSON != tt.expectedJSON {
				t.Errorf("expected log JSON %t, but got %t", tt.expectedJSON, config.Log.JSON)
			}
		})
	}
}
