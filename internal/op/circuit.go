package op

import (
	"context"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"gorm.io/gorm/clause"
)

// CircuitStateUpsert persists circuit breaker state to database.
func CircuitStateUpsert(ctx context.Context, state model.CircuitBreakerState) error {
	return db.GetDB().WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&state).Error
}

// CircuitStateList loads all persisted circuit breaker states.
func CircuitStateList(ctx context.Context) ([]model.CircuitBreakerState, error) {
	var states []model.CircuitBreakerState
	if err := db.GetDB().WithContext(ctx).Find(&states).Error; err != nil {
		return nil, err
	}
	return states, nil
}

// CircuitInit restores circuit breaker states into memory.
func CircuitInit(ctx context.Context) error {
	states, err := CircuitStateList(ctx)
	if err != nil {
		return err
	}
	balancer.RestoreFromDB(states)
	return nil
}
