package graph

import (
	"scoring_api_gateway/internal/service"

	"go.uber.org/zap"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	VerificationService service.VerificationService
	Logger              *zap.Logger
}
