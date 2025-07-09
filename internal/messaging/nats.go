package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"scoring_api_gateway/graph/model"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type NATSClient interface {
	PublishVerificationRequest(ctx context.Context, verification *model.Verification) error
	SubscribeToVerificationCompleted(ctx context.Context, handler func(*model.Verification)) error
	Close()
}

type natsClient struct {
	conn   *nats.Conn
	logger *zap.Logger
}

func NewNATSClient(url string, logger *zap.Logger) (NATSClient, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("connected to NATS", zap.String("url", url))
	return &natsClient{
		conn:   conn,
		logger: logger,
	}, nil
}

type CreateVerificationMessage struct {
	VerificationID string                       `json:"verification_id"`
	INN            string                       `json:"inn"`
	RequestedTypes []model.VerificationDataType `json:"requested_types"`
	AuthorEmail    string                       `json:"author_email"`
}

type VerificationCompletedMessage struct {
	VerificationID string `json:"verification_id"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
}

func (c *natsClient) PublishVerificationRequest(ctx context.Context, verification *model.Verification) error {
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

func (c *natsClient) SubscribeToVerificationCompleted(ctx context.Context, handler func(*model.Verification)) error {
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

func (c *natsClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
}
