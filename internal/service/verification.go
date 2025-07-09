package service

import (
	"context"
	"fmt"

	"scoring_api_gateway/graph/model"
	"scoring_api_gateway/internal/messaging"
	"scoring_api_gateway/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type VerificationService interface {
	CreateVerification(ctx context.Context, inn string, requestedTypes []model.VerificationDataType, authorEmail string) (*model.Verification, error)
	GetVerification(ctx context.Context, id string) (*model.Verification, error)
	GetAllVerifications(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error)
	GetVerificationWithData(ctx context.Context, id string) (*model.VerificationDataResult, error)
}

type verificationService struct {
	repo   repository.VerificationRepository
	nats   messaging.NATSClient
	logger *zap.Logger
}

func NewVerificationService(repo repository.VerificationRepository, nats messaging.NATSClient, logger *zap.Logger) VerificationService {
	return &verificationService{
		repo:   repo,
		nats:   nats,
		logger: logger,
	}
}

func (s *verificationService) CreateVerification(ctx context.Context, inn string, requestedTypes []model.VerificationDataType, authorEmail string) (*model.Verification, error) {
	if inn == "" {
		return nil, fmt.Errorf("inn cannot be empty")
	}

	if len(requestedTypes) == 0 {
		return nil, fmt.Errorf("at least one data type must be requested")
	}

	if len(inn) != 10 && len(inn) != 12 {
		return nil, fmt.Errorf("inn must be 10 or 12 digits, got %d", len(inn))
	}

	verificationID := uuid.New().String()

	verification := &model.Verification{
		ID:                 verificationID,
		Inn:                inn,
		Status:             model.VerificationStatusInProcess,
		AuthorEmail:        authorEmail,
		RequestedDataTypes: requestedTypes,
	}

	err := s.nats.PublishVerificationRequest(ctx, verification)
	if err != nil {
		s.logger.Error("failed to publish verification request", zap.Error(err), zap.String("verification_id", verificationID))
		return nil, fmt.Errorf("failed to publish verification request: %w", err)
	}

	s.logger.Info("verification request published", zap.String("verification_id", verificationID), zap.String("inn", inn))
	return verification, nil
}

func (s *verificationService) GetVerification(ctx context.Context, id string) (*model.Verification, error) {
	if id == "" {
		return nil, fmt.Errorf("verification id cannot be empty")
	}

	verification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get verification from repository", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	if verification == nil {
		return nil, fmt.Errorf("verification not found: %s", id)
	}

	return verification, nil
}

func (s *verificationService) GetAllVerifications(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error) {
	if limit != nil && *limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative, got %d", *limit)
	}

	if offset != nil && *offset < 0 {
		return nil, fmt.Errorf("offset must be non-negative, got %d", *offset)
	}

	return s.repo.GetAll(ctx, limit, offset)
}

func (s *verificationService) GetVerificationWithData(ctx context.Context, id string) (*model.VerificationDataResult, error) {
	if id == "" {
		return nil, fmt.Errorf("verification id cannot be empty")
	}

	verification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get verification from repository", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	if verification == nil {
		return nil, fmt.Errorf("verification not found: %s", id)
	}

	// Создаем результат и маппим данные по типам
	result := &model.VerificationDataResult{
		Verification: verification,
	}

	// Маппим данные по типам
	for _, data := range verification.Data {
		switch data.DataType {
		case model.VerificationDataTypeBasicInformation:
			result.BasicInformation = &data.Data
		case model.VerificationDataTypeActivities:
			result.Activities = &data.Data
		case model.VerificationDataTypeAddressesByCredinform:
			result.AddressesByCredinform = &data.Data
		case model.VerificationDataTypeAddressesByUnifiedStateRegister:
			result.AddressesByUnifiedStateRegister = &data.Data
		case model.VerificationDataTypeAffiliatedCompanies:
			result.AffiliatedCompanies = &data.Data
		case model.VerificationDataTypeArbitrageStatistics:
			result.ArbitrageStatistics = &data.Data
		}
	}

	return result, nil
}
