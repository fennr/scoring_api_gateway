package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Интерфейс для pgxpool.Pool
type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Mock для pgxpool.Pool
type mockDBPool struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *mockDBPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return nil
}

// Mock для pgx.Row
type mockRow struct {
	scanFunc func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	return nil
}

// Тестовая версия dataCacheRepository
type testDataCacheRepository struct {
	db     dbPool
	logger *zap.Logger
}

func (r *testDataCacheRepository) GetDataByHash(ctx context.Context, hash string) (string, error) {
	query := `SELECT data FROM verification_data_cache WHERE data_hash = $1`

	var data string
	err := r.db.QueryRow(ctx, query, hash).Scan(&data)
	if err != nil {
		r.logger.Error("data not found in cache", zap.String("hash", hash), zap.Error(err))
		return "", fmt.Errorf("data not found in cache for hash %s: %w", hash, err)
	}

	r.logger.Debug("data retrieved from cache", zap.String("hash", hash))
	return data, nil
}

func TestGetDataByHash(t *testing.T) {
	tests := []struct {
		name          string
		hash          string
		mockData      string
		mockError     error
		expectedData  string
		expectedError string
	}{
		{
			name:          "successful_get",
			hash:          "abc123",
			mockData:      `{"test": "data"}`,
			mockError:     nil,
			expectedData:  `{"test": "data"}`,
			expectedError: "",
		},
		{
			name:          "data_not_found",
			hash:          "nonexistent",
			mockData:      "",
			mockError:     pgx.ErrNoRows,
			expectedData:  "",
			expectedError: "data not found in cache for hash nonexistent",
		},
		{
			name:          "database_error",
			hash:          "error_hash",
			mockData:      "",
			mockError:     errors.New("database connection failed"),
			expectedData:  "",
			expectedError: "data not found in cache for hash error_hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool := &mockDBPool{
				queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							if tt.mockError != nil {
								return tt.mockError
							}
							if len(dest) > 0 {
								if strPtr, ok := dest[0].(*string); ok {
									*strPtr = tt.mockData
								}
							}
							return nil
						},
					}
				},
			}

			logger := zaptest.NewLogger(t)
			repo := &testDataCacheRepository{
				db:     mockPool,
				logger: logger,
			}

			data, err := repo.GetDataByHash(context.Background(), tt.hash)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', but got nil", tt.expectedError)
					return
				}
				if !containsError(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', but got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if data != tt.expectedData {
				t.Errorf("expected data '%s', but got '%s'", tt.expectedData, data)
			}
		})
	}
}

// Вспомогательная функция для проверки содержания ошибки
func containsError(got, want string) bool {
	return len(got) > 0 && len(want) > 0 && (got == want ||
		(len(got) >= len(want) && got[:len(want)] == want))
}
