package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DataCacheRepository interface {
	GetDataByHash(ctx context.Context, hash string) (string, error)
}

type dataCacheRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewDataCacheRepository(db *pgxpool.Pool, logger *zap.Logger) DataCacheRepository {
	return &dataCacheRepository{
		db:     db,
		logger: logger,
	}
}

// GetDataByHash получает данные из кэша по хэшу
func (r *dataCacheRepository) GetDataByHash(ctx context.Context, hash string) (string, error) {
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
