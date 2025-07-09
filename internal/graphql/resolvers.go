package graphql

import (
	"context"

	"scoring_api_gateway/graph/model"
	"scoring_api_gateway/internal/service"

	"go.uber.org/zap"
)

type Resolver struct {
	verificationService service.VerificationService
	logger              *zap.Logger
}

func NewResolver(verificationService service.VerificationService, logger *zap.Logger) *Resolver {
	return &Resolver{
		verificationService: verificationService,
		logger:              logger,
	}
}

func (r *Resolver) Query() interface{} {
	return &queryResolver{r}
}

func (r *Resolver) Mutation() interface{} {
	return &mutationResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Verification(ctx context.Context, id string) (*model.Verification, error) {
	r.logger.Info("query verification", zap.String("id", id))

	verification, err := r.verificationService.GetVerification(ctx, id)
	if err != nil {
		r.logger.Error("failed to get verification", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	return verification, nil
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateVerification(ctx context.Context, inn string, requestedDataTypes []model.VerificationDataType) (*model.Verification, error) {
	r.logger.Info("create verification", zap.String("inn", inn), zap.Any("requested_types", requestedDataTypes))

	authorEmail := "test@example.com" // TODO: получить из контекста аутентификации

	verification, err := r.verificationService.CreateVerification(ctx, inn, requestedDataTypes, authorEmail)
	if err != nil {
		r.logger.Error("failed to create verification", zap.Error(err), zap.String("inn", inn))
		return nil, err
	}

	return verification, nil
}
