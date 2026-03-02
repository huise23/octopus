package model

import "time"

// CircuitBreakerState stores circuit breaker state for persistence.
type CircuitBreakerState struct {
	ChannelID         int             `json:"channel_id" gorm:"primaryKey;autoIncrement:false"`
	ChannelKeyID      int             `json:"channel_key_id" gorm:"primaryKey;autoIncrement:false"`
	ModelName         string          `json:"model_name" gorm:"primaryKey;size:64"`
	State             int             `json:"state"`
	Failures          int64           `json:"failures" gorm:"bigint"`
	RateLimitFailures int64           `json:"rate_limit_failures" gorm:"bigint"`
	LastFailureTime   time.Time       `json:"last_failure_time"`
	FailedDays        map[string]bool `json:"failed_days" gorm:"serializer:json"`
	PermanentBlock    bool            `json:"permanent_block"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (CircuitBreakerState) TableName() string {
	return "circuit_breaker_state"
}
