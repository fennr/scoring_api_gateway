package types

import "time"

// VerificationDataCache представляет запись в таблице verification_data_cache
type VerificationDataCache struct {
	ID        string    `json:"id" db:"id"`
	DataHash  string    `json:"data_hash" db:"data_hash"`
	Data      string    `json:"data" db:"data"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// VerificationDataWithHash представляет запись в verification_data с хэшем
type VerificationDataWithHash struct {
	ID             string    `json:"id" db:"id"`
	VerificationID string    `json:"verification_id" db:"verification_id"`
	DataType       string    `json:"data_type" db:"data_type"`
	Data           *string   `json:"data,omitempty" db:"data"` // Для обратной совместимости
	DataHash       *string   `json:"data_hash,omitempty" db:"data_hash"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
