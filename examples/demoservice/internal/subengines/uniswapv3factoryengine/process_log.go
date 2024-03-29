package uniswapv3factoryengine

import (
	"reflect"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/pkg/logger"

	"github.com/soyart/superwatcher/examples/demoservice/internal/domain/entity"
)

func (e *uniswapv3PoolFactoryEngine) handleGoodLog(log *types.Log) (PoolFactoryArtifact, error) {
	artifact := make(PoolFactoryArtifact)
	logEventKey := log.Topics[0]

	for _, event := range e.poolFactoryContract.ContractEvents {
		// This engine is supposed to handle more than 1 event,
		// but it's not yet finished now.
		if logEventKey == event.ID || event.Name == "PoolCreated" {
			pool, err := mapLogToPoolCreated(e.poolFactoryContract.ContractABI, event.Name, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to map PoolCreated log to domain struct")
			}
			if pool == nil {
				logger.Panic("nil pool mapped - should not happen")
			}
			if err := e.handlePoolCreated(pool); err != nil {
				return nil, errors.Wrap(err, "failed to process poolCreated")
			}

			// Saves engine artifact
			artifact[*pool] = PoolFactoryStateCreated
		}
	}

	return artifact, nil
}

func (e *uniswapv3PoolFactoryEngine) handleReorgedLog(log *types.Log, artifacts []superwatcher.Artifact) (PoolFactoryArtifact, error) {
	logEventKey := log.Topics[0]

	// Find poolFactory artifact here
	var poolArtifact PoolFactoryArtifact
	for _, artifacts := range artifacts {
		pa, ok := artifacts.(PoolFactoryArtifact)
		if ok {
			poolArtifact = pa
			continue
		}

		if !ok {
			for _, artifact := range artifacts.([]superwatcher.Artifact) {
				pa, ok := artifact.(PoolFactoryArtifact)
				if !ok {
					e.debugger.Debug(1, "found non-pool artifact", zap.String("actual type", reflect.TypeOf(artifact).String()))
					continue
				}

				poolArtifact = pa
			}
		}
	}

	var returnArtifact PoolFactoryArtifact
	for _, event := range e.poolFactoryContract.ContractEvents {
		// This engine is supposed to handle more than 1 event,
		// but it's not yet finished now.
		if logEventKey == event.ID || event.Name == "PoolCreated" {
			pool, err := mapLogToPoolCreated(e.poolFactoryContract.ContractABI, event.Name, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to map PoolCreated log to domain struct")
			}

			returnArtifact, err = e.handleReorgedPool(pool, poolArtifact)
			if err != nil {
				return nil, errors.Wrap(err, "failed to handle reorged PoolCreated")
			}
		}

		continue
	}

	return returnArtifact, nil
}

// In uniswapv3poolfactory case, we only revert PoolCreated in the db.
// Other service may need more elaborate HandleReorg.
func (e *uniswapv3PoolFactoryEngine) handleReorgedPool(
	pool *entity.Uniswapv3PoolCreated,
	poolArtifact PoolFactoryArtifact,
) (
	PoolFactoryArtifact,
	error,
) {
	poolState := poolArtifact[*pool]

	switch poolState { //nolint:gocritic
	case PoolFactoryStateCreated:
		if err := e.revertPoolCreated(pool); err != nil {
			return nil, errors.Wrapf(err, "failed to revert poolCreated for pool %s", pool.Address.String())
		}
	}

	poolArtifact[*pool] = PoolFactoryStateNull
	return poolArtifact, nil
}
