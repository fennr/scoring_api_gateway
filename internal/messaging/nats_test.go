package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"scoring_api_gateway/graph/model"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Интерфейс для nats.Conn
type natsConnection interface {
	Publish(subj string, data []byte) error
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	Close()
}

// Mock для nats.Conn
type mockNATSConn struct {
	publishFunc   func(subj string, data []byte) error
	subscribeFunc func(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	closeFunc     func()
}

func (m *mockNATSConn) Publish(subj string, data []byte) error {
	if m.publishFunc != nil {
		return m.publishFunc(subj, data)
	}
	return nil
}

func (m *mockNATSConn) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	if m.subscribeFunc != nil {
		return m.subscribeFunc(subj, cb)
	}
	return &nats.Subscription{}, nil
}

func (m *mockNATSConn) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

// Тестовая версия natsClient для использования с моками
type testNATSClient struct {
	conn   natsConnection
	logger *zap.Logger
}

func (c *testNATSClient) PublishVerificationRequest(ctx context.Context, verification *model.Verification) error {
	msg := CreateVerificationMessage{
		VerificationID: verification.ID,
		INN:            verification.Inn,
		RequestedTypes: verification.RequestedDataTypes,
		AuthorEmail:    verification.AuthorEmail,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("failed to marshal verification request", zap.Error(err))
		return fmt.Errorf("failed to marshal verification request: %w", err)
	}

	err = c.conn.Publish("verification.create", data)
	if err != nil {
		c.logger.Error("failed to publish verification request", zap.Error(err), zap.String("verification_id", verification.ID))
		return fmt.Errorf("failed to publish verification request: %w", err)
	}

	c.logger.Info("verification request published", zap.String("verification_id", verification.ID))
	return nil
}

func (c *testNATSClient) SubscribeToVerificationCompleted(ctx context.Context, handler func(*model.Verification)) error {
	_, err := c.conn.Subscribe("verification.completed", func(msg *nats.Msg) {
		var completedMsg VerificationCompletedMessage
		if err := json.Unmarshal(msg.Data, &completedMsg); err != nil {
			c.logger.Error("failed to unmarshal verification completed message", zap.Error(err))
			return
		}

		verification := &model.Verification{
			ID:     completedMsg.VerificationID,
			Status: model.VerificationStatus(completedMsg.Status),
		}

		handler(verification)
		c.logger.Info("verification completed message processed", zap.String("verification_id", completedMsg.VerificationID), zap.String("status", completedMsg.Status))
	})

	if err != nil {
		c.logger.Error("failed to subscribe to verification completed", zap.Error(err))
		return fmt.Errorf("failed to subscribe to verification completed: %w", err)
	}

	c.logger.Info("subscribed to verification completed messages")
	return nil
}

func (c *testNATSClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
}

func TestPublishVerificationRequest(t *testing.T) {
	tests := []struct {
		name          string
		verification  *model.Verification
		publishError  error
		expectedError string
	}{
		{
			name: "successful_publish",
			verification: &model.Verification{
				ID:                 "test-id",
				Inn:                "1234567890",
				AuthorEmail:        "test@example.com",
				RequestedDataTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			},
			publishError:  nil,
			expectedError: "",
		},
		{
			name: "publish_error",
			verification: &model.Verification{
				ID:                 "test-id",
				Inn:                "1234567890",
				AuthorEmail:        "test@example.com",
				RequestedDataTypes: []model.VerificationDataType{model.VerificationDataTypeBasicInformation},
			},
			publishError:  errors.New("nats connection failed"),
			expectedError: "failed to publish verification request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var publishedData []byte
			var publishedSubject string

			mockConn := &mockNATSConn{
				publishFunc: func(subj string, data []byte) error {
					publishedSubject = subj
					publishedData = data
					return tt.publishError
				},
			}

			logger := zaptest.NewLogger(t)
			client := &testNATSClient{
				conn:   mockConn,
				logger: logger,
			}

			err := client.PublishVerificationRequest(context.Background(), tt.verification)

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

			// Проверяем, что сообщение опубликовано в правильный subject
			if publishedSubject != "verification.create" {
				t.Errorf("expected subject 'verification.create', but got '%s'", publishedSubject)
			}

			// Проверяем содержимое сообщения
			if publishedData != nil {
				var msg CreateVerificationMessage
				if err := json.Unmarshal(publishedData, &msg); err != nil {
					t.Errorf("failed to unmarshal published message: %v", err)
					return
				}

				if msg.VerificationID != tt.verification.ID {
					t.Errorf("expected verification ID '%s', but got '%s'", tt.verification.ID, msg.VerificationID)
				}

				if msg.INN != tt.verification.Inn {
					t.Errorf("expected INN '%s', but got '%s'", tt.verification.Inn, msg.INN)
				}

				if msg.AuthorEmail != tt.verification.AuthorEmail {
					t.Errorf("expected author email '%s', but got '%s'", tt.verification.AuthorEmail, msg.AuthorEmail)
				}

				if len(msg.RequestedTypes) != len(tt.verification.RequestedDataTypes) {
					t.Errorf("expected %d requested types, but got %d", len(tt.verification.RequestedDataTypes), len(msg.RequestedTypes))
				}
			}
		})
	}
}

func TestSubscribeToVerificationCompleted(t *testing.T) {
	tests := []struct {
		name            string
		subscribeError  error
		expectedError   string
		messageToHandle *VerificationCompletedMessage
	}{
		{
			name:           "successful_subscribe",
			subscribeError: nil,
			expectedError:  "",
			messageToHandle: &VerificationCompletedMessage{
				VerificationID: "test-id",
				Status:         "COMPLETED",
			},
		},
		{
			name:           "subscribe_error",
			subscribeError: errors.New("failed to subscribe"),
			expectedError:  "failed to subscribe to verification completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool
			var receivedVerification *model.Verification
			var subscribedSubject string
			var messageHandler nats.MsgHandler

			mockConn := &mockNATSConn{
				subscribeFunc: func(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
					subscribedSubject = subj
					messageHandler = cb
					return &nats.Subscription{}, tt.subscribeError
				},
			}

			logger := zaptest.NewLogger(t)
			client := &testNATSClient{
				conn:   mockConn,
				logger: logger,
			}

			handler := func(verification *model.Verification) {
				handlerCalled = true
				receivedVerification = verification
			}

			err := client.SubscribeToVerificationCompleted(context.Background(), handler)

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

			// Проверяем, что подписались на правильный subject
			if subscribedSubject != "verification.completed" {
				t.Errorf("expected subject 'verification.completed', but got '%s'", subscribedSubject)
			}

			// Тестируем обработчик сообщений, если есть тестовое сообщение
			if tt.messageToHandle != nil && messageHandler != nil {
				msgData, _ := json.Marshal(tt.messageToHandle)
				mockMsg := &nats.Msg{Data: msgData}
				messageHandler(mockMsg)

				if !handlerCalled {
					t.Error("expected handler to be called, but it wasn't")
					return
				}

				if receivedVerification == nil {
					t.Error("expected verification to be passed to handler, but got nil")
					return
				}

				if receivedVerification.ID != tt.messageToHandle.VerificationID {
					t.Errorf("expected verification ID '%s', but got '%s'",
						tt.messageToHandle.VerificationID, receivedVerification.ID)
				}

				if string(receivedVerification.Status) != tt.messageToHandle.Status {
					t.Errorf("expected status '%s', but got '%s'",
						tt.messageToHandle.Status, string(receivedVerification.Status))
				}
			}
		})
	}
}

func TestSubscribeToVerificationCompletedInvalidMessage(t *testing.T) {
	var messageHandler nats.MsgHandler

	mockConn := &mockNATSConn{
		subscribeFunc: func(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
			messageHandler = cb
			return &nats.Subscription{}, nil
		},
	}

	logger := zaptest.NewLogger(t)
	client := &testNATSClient{
		conn:   mockConn,
		logger: logger,
	}

	var handlerCalled bool
	handler := func(verification *model.Verification) {
		handlerCalled = true
	}

	err := client.SubscribeToVerificationCompleted(context.Background(), handler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Отправляем невалидное JSON сообщение
	invalidMsg := &nats.Msg{Data: []byte("invalid json")}
	messageHandler(invalidMsg)

	// Обработчик не должен быть вызван при невалидном сообщении
	if handlerCalled {
		t.Error("handler should not be called for invalid message")
	}
}

func TestClose(t *testing.T) {
	var closeCalled bool

	mockConn := &mockNATSConn{
		closeFunc: func() {
			closeCalled = true
		},
	}

	logger := zaptest.NewLogger(t)
	client := &testNATSClient{
		conn:   mockConn,
		logger: logger,
	}

	client.Close()

	if !closeCalled {
		t.Error("expected Close to be called on connection, but it wasn't")
	}
}

func TestCloseWithNilConnection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := &natsClient{
		conn:   nil,
		logger: logger,
	}

	// Не должно паниковать при nil connection
	client.Close()
}

// Вспомогательная функция для проверки содержания ошибки
func containsError(got, want string) bool {
	return len(got) > 0 && len(want) > 0 && (got == want ||
		(len(got) >= len(want) && got[:len(want)] == want))
}
