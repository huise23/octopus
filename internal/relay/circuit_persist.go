package relay

import (
	"context"

	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/utils/log"
)

func persistCircuitState(ctx context.Context, channelID, keyID int, modelName string) {
	state, ok := balancer.GetStateSnapshot(channelID, keyID, modelName)
	if !ok {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := op.CircuitStateUpsert(ctx, state); err != nil {
		log.Warnf("persist circuit state failed: %v", err)
	}
}
