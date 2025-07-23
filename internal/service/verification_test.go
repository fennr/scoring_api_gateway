package service

import (
	"context"
	"errors"
	"testing"

	"scoring_api_gateway/graph/model"

	"go.uber.org/zap/zaptest"
)

// Mock для VerificationRepository
type mockVerificationRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*model.Verification, error)
	getAllFunc  func(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error)
}

func (m *mockVerificationRepository) GetByID(ctx context.Context, id string) (*model.Verification, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockVerificationRepository) GetAll(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, limit, offset)
	}
	return nil, nil
}

// Mock для NATSClient
type mockNATSClient struct {
	publishVerificationRequestFunc   func(ctx context.Context, verification *model.Verification) error
	subscribeToVerificationCompleted func(ctx context.Context, handler func(*model.Verification)) error
	closeFunc                        func()
}

func (m *mockNATSClient) PublishVerificationRequest(ctx context.Context, verification *model.Verification) error {
	if m.publishVerificationRequestFunc != nil {
		return m.publishVerificationRequestFunc(ctx, verification)
	}
	return nil
}

func (m *mockNATSClient) SubscribeToVerificationCompleted(ctx context.Context, handler func(*model.Verification)) error {
	if m.subscribeToVerificationCompleted != nil {
		return m.subscribeToVerificationCompleted(ctx, handler)
	}
	return nil
}

func (m *mockNATSClient) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

func TestCreateVerification(t *testing.T) {
	tests := []struct {
		name           string
		inn            string
		requestedTypes []model.VerificationDataType
		authorEmail    string
		publishError   error
		expectedError  string
	}{
		{
			name:           "successful_creation",
			inn:            "1234567890",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "",
		},
		{
			name:           "empty_inn",
			inn:            "",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "inn cannot be empty",
		},
		{
			name:           "invalid_inn_length_short",
			inn:            "123",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "inn must be 10 or 12 digits, got 3",
		},
		{
			name:           "invalid_inn_length_long",
			inn:            "12345678901234",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "inn must be 10 or 12 digits, got 14",
		},
		{
			name:           "valid_inn_12_digits",
			inn:            "123456789012",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "",
		},
		{
			name:           "empty_requested_types",
			inn:            "1234567890",
			requestedTypes: []model.VerificationDataType{},
			authorEmail:    "test@example.com",
			publishError:   nil,
			expectedError:  "at least one data type must be requested",
		},
		{
			name:           "nats_publish_error",
			inn:            "1234567890",
			requestedTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			authorEmail:    "test@example.com",
			publishError:   errors.New("nats connection failed"),
			expectedError:  "failed to publish verification request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockVerificationRepository{}
			mockNATS := &mockNATSClient{
				publishVerificationRequestFunc: func(ctx context.Context, verification *model.Verification) error {
					return tt.publishError
				},
			}
			logger := zaptest.NewLogger(t)

			service := NewVerificationService(mockRepo, mockNATS, logger)

			verification, err := service.CreateVerification(context.Background(), tt.inn, tt.requestedTypes, tt.authorEmail)

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

			if verification == nil {
				t.Error("expected verification to be created, but got nil")
				return
			}

			if verification.Inn != tt.inn {
				t.Errorf("expected inn '%s', but got '%s'", tt.inn, verification.Inn)
			}

			if verification.AuthorEmail != tt.authorEmail {
				t.Errorf("expected author email '%s', but got '%s'", tt.authorEmail, verification.AuthorEmail)
			}

			if verification.Status != model.VerificationStatusInProcess {
				t.Errorf("expected status '%s', but got '%s'", model.VerificationStatusInProcess, verification.Status)
			}

			if len(verification.RequestedDataTypes) != len(tt.requestedTypes) {
				t.Errorf("expected %d requested types, but got %d", len(tt.requestedTypes), len(verification.RequestedDataTypes))
			}
		})
	}
}

func TestGetVerification(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		repoResult    *model.Verification
		repoError     error
		expectedError string
	}{
		{
			name: "successful_get",
			id:   "test-id",
			repoResult: &model.Verification{
				ID:     "test-id",
				Inn:    "1234567890",
				Status: model.VerificationStatusCompleted,
			},
			repoError:     nil,
			expectedError: "",
		},
		{
			name:          "empty_id",
			id:            "",
			repoResult:    nil,
			repoError:     nil,
			expectedError: "verification id cannot be empty",
		},
		{
			name:          "verification_not_found",
			id:            "non-existent-id",
			repoResult:    nil,
			repoError:     nil,
			expectedError: "verification not found: non-existent-id",
		},
		{
			name:          "repository_error",
			id:            "test-id",
			repoResult:    nil,
			repoError:     errors.New("database connection failed"),
			expectedError: "failed to get verification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockVerificationRepository{
				getByIDFunc: func(ctx context.Context, id string) (*model.Verification, error) {
					return tt.repoResult, tt.repoError
				},
			}
			mockNATS := &mockNATSClient{}
			logger := zaptest.NewLogger(t)

			service := NewVerificationService(mockRepo, mockNATS, logger)

			verification, err := service.GetVerification(context.Background(), tt.id)

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

			if verification == nil {
				t.Error("expected verification, but got nil")
				return
			}

			if verification.ID != tt.repoResult.ID {
				t.Errorf("expected ID '%s', but got '%s'", tt.repoResult.ID, verification.ID)
			}
		})
	}
}

func TestGetAllVerifications(t *testing.T) {
	tests := []struct {
		name          string
		limit         *int32
		offset        *int32
		repoResult    []*model.Verification
		repoError     error
		expectedError string
	}{
		{
			name:   "successful_get_all",
			limit:  int32Ptr(10),
			offset: int32Ptr(0),
			repoResult: []*model.Verification{
				{ID: "1", Inn: "1234567890"},
				{ID: "2", Inn: "0987654321"},
			},
			repoError:     nil,
			expectedError: "",
		},
		{
			name:          "negative_limit",
			limit:         int32Ptr(-1),
			offset:        int32Ptr(0),
			repoResult:    nil,
			repoError:     nil,
			expectedError: "limit must be non-negative, got -1",
		},
		{
			name:          "negative_offset",
			limit:         int32Ptr(10),
			offset:        int32Ptr(-5),
			repoResult:    nil,
			repoError:     nil,
			expectedError: "offset must be non-negative, got -5",
		},
		{
			name:       "nil_limit_and_offset",
			limit:      nil,
			offset:     nil,
			repoResult: []*model.Verification{},
			repoError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockVerificationRepository{
				getAllFunc: func(ctx context.Context, limit *int32, offset *int32) ([]*model.Verification, error) {
					return tt.repoResult, tt.repoError
				},
			}
			mockNATS := &mockNATSClient{}
			logger := zaptest.NewLogger(t)

			service := NewVerificationService(mockRepo, mockNATS, logger)

			verifications, err := service.GetAllVerifications(context.Background(), tt.limit, tt.offset)

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

			if len(verifications) != len(tt.repoResult) {
				t.Errorf("expected %d verifications, but got %d", len(tt.repoResult), len(verifications))
			}
		})
	}
}

func TestGetVerificationWithData(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		repoResult    *model.Verification
		repoError     error
		expectedError string
	}{
		{
			name: "successful_get_with_data",
			id:   "test-id",
			repoResult: &model.Verification{
				ID:     "test-id",
				Inn:    "1234567890",
				Status: model.VerificationStatusCompleted,
				Data: []*model.VerificationData{
					{
						DataType: model.VerificationDataTypeBasicInformation,
						Data:     `{"name": "Test Company"}`,
					},
					{
						DataType: model.VerificationDataTypeActivities,
						Data:     `{"activities": []}`,
					},
				},
			},
			repoError:     nil,
			expectedError: "",
		},
		{
			name:          "empty_id",
			id:            "",
			repoResult:    nil,
			repoError:     nil,
			expectedError: "verification id cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockVerificationRepository{
				getByIDFunc: func(ctx context.Context, id string) (*model.Verification, error) {
					return tt.repoResult, tt.repoError
				},
			}
			mockNATS := &mockNATSClient{}
			logger := zaptest.NewLogger(t)

			service := NewVerificationService(mockRepo, mockNATS, logger)

			result, err := service.GetVerificationWithData(context.Background(), tt.id)

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

			if result == nil {
				t.Error("expected result, but got nil")
				return
			}

			if result.Verification.ID != tt.repoResult.ID {
				t.Errorf("expected verification ID '%s', but got '%s'", tt.repoResult.ID, result.Verification.ID)
			}

			// Проверяем маппинг данных по типам
			if tt.repoResult.Data != nil {
				for _, data := range tt.repoResult.Data {
					switch data.DataType {
					case model.VerificationDataTypeBasicInformation:
						if result.BasicInformation == nil || *result.BasicInformation != data.Data {
							t.Errorf("expected basic information data '%s', but got '%v'", data.Data, result.BasicInformation)
						}
					case model.VerificationDataTypeActivities:
						if result.Activities == nil || *result.Activities != data.Data {
							t.Errorf("expected activities data '%s', but got '%v'", data.Data, result.Activities)
						}
					}
				}
			}
		})
	}
}

// Вспомогательная функция для создания указателя на int32
func int32Ptr(i int32) *int32 {
	return &i
}

// Вспомогательная функция для проверки содержания ошибки
func containsError(got, want string) bool {
	return len(got) > 0 && len(want) > 0 && (got == want ||
		(len(got) >= len(want) && got[:len(want)] == want))
}
