package repository

import (
	"context"
	"fmt"
	"time"

	"scoring_api_gateway/graph/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type VerificationRepository interface {
	GetByID(ctx context.Context, id string) (*model.Verification, error)
	GetAll(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error)
}

type verificationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewVerificationRepository(db *pgxpool.Pool, logger *zap.Logger) VerificationRepository {
	return &verificationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *verificationRepository) GetByID(ctx context.Context, id string) (*model.Verification, error) {
	query := `
		SELECT id, inn, status, author_email, company_id, requested_data_types, created_at, updated_at
		FROM verifications
		WHERE id = $1
	`

	var verification model.Verification
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, query, id).
		Scan(&verification.ID, &verification.Inn, &verification.Status, &verification.AuthorEmail, &verification.CompanyID, &verification.RequestedDataTypes, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("failed to get verification", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	verification.CreatedAt = createdAt.Format(time.RFC3339)
	verification.UpdatedAt = updatedAt.Format(time.RFC3339)

	dataQuery := `
		SELECT data_type, data, created_at
		FROM verification_data
		WHERE verification_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, dataQuery, id)
	if err != nil {
		r.logger.Error("failed to get verification data", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get verification data: %w", err)
	}
	defer rows.Close()

	var data []*model.VerificationData
	for rows.Next() {
		var vd model.VerificationData
		var dataCreatedAt time.Time
		err := rows.Scan(&vd.DataType, &vd.Data, &dataCreatedAt)
		if err != nil {
			r.logger.Error("failed to scan verification data", zap.Error(err))
			continue
		}
		vd.CreatedAt = dataCreatedAt.Format(time.RFC3339)
		data = append(data, &vd)
	}

	verification.Data = data
	return &verification, nil
}

func (r *verificationRepository) GetAll(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error) {
	query := `
		SELECT id, inn, status, author_email, company_id, requested_data_types, created_at, updated_at
		FROM verifications
		ORDER BY created_at DESC
	`

	if limit != nil || offset != nil {
		if limit != nil && offset != nil {
			query += fmt.Sprintf(" LIMIT %d OFFSET %d", *limit, *offset)
		} else if limit != nil {
			query += fmt.Sprintf(" LIMIT %d", *limit)
		} else if offset != nil {
			query += fmt.Sprintf(" OFFSET %d", *offset)
		}
	}

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to get all verifications", zap.Error(err))
		return nil, fmt.Errorf("failed to get all verifications: %w", err)
	}
	defer rows.Close()

	var verifications []*model.Verification
	for rows.Next() {
		var v model.Verification
		var createdAt, updatedAt time.Time
		err := rows.Scan(&v.ID, &v.Inn, &v.Status, &v.AuthorEmail, &v.CompanyID, &v.RequestedDataTypes, &createdAt, &updatedAt)
		if err != nil {
			r.logger.Error("failed to scan verification", zap.Error(err))
			continue
		}
		v.CreatedAt = createdAt.Format(time.RFC3339)
		v.UpdatedAt = updatedAt.Format(time.RFC3339)
		verifications = append(verifications, &v)
	}

	return verifications, nil
}
