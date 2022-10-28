package ensengine

// TODO: Fix engine.Artifact

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher/domain/usecase/engine"
	"github.com/artnoi43/superwatcher/lib/logger"
	"github.com/artnoi43/superwatcher/superwatcher-demo/domain/entity"
)

// MapLogToItem wraps mapLogToItem, so the latter can be unit tested.
func (e *ensEngine) HandleGoodLogs(
	logs []*types.Log,
	artifacts []engine.Artifact, // Ignore
) (
	[]engine.Artifact,
	error,
) {
	logger.Debug("poolfactory.HandleGoodLog", zap.Any("input artifacts", artifacts))

	var err error
	for _, log := range logs {
		_, err = e.HandleGoodLog(log)
		if err != nil {
			return nil, errors.Wrapf(err, "HandleGoodLog failed on log txHash %s", log.BlockHash.String())
		}
	}

	return nil, errors.New("not implemented")
}

func (e *ensEngine) HandleGoodLog(log *types.Log) (any, error) {
	logEventKey := log.Topics[0]

	for _, event := range e.ensContract.ContractEvents {
		// This engine is supposed to handle more than 1 event,
		// but it's not yet finished now.
		if logEventKey == event.ID {
			switch event.Name {
			case "NameRegistered": // New domain registered
			case "Transfer": // Logged when the owner of a node transfers ownership to a new account.
			}
		}
	}

	return nil, errors.New("not implemented")
}

func (e *ensEngine) HandleReorgedLogs(logs []*types.Log, artifacts []engine.Artifact) ([]engine.Artifact, error) {
	logger.Debug("poolfactory.HandleReorgedLogs", zap.Any("input artifacts", artifacts))

	var ensDomains []entity.ENS

	for _, log := range logs {
		ens, err := e.handleReorgedLog(log, artifacts)
		if err != nil {
			return nil, errors.Wrap(err, "ensEngine.handleReorgedLog failed")
		}

		ensDomains = append(ensDomains, ens)
	}

	return []engine.Artifact{ensDomains}, nil
}

func (e *ensEngine) handleReorgedLog(log *types.Log, artifacts []engine.Artifact) (entity.ENS, error) {

	var ens entity.ENS
	for _, a := range artifacts {
		pa, ok := a.(entity.ENS)
		if !ok {
			logger.Debug("found non-pool artifact")
			continue
		}

		ens = pa
	}

	logEventKey := log.Topics[0]
	for _, event := range e.ensContract.ContractEvents {
		// This engine is supposed to handle more than 1 event,
		// but it's not yet finished now.
		if logEventKey == event.ID {
			switch event.Name {

			}
		}
	}

	return ens, fmt.Errorf("event topic %s not found", logEventKey)
}

// Unused by this service
func (e *ensEngine) HandleEmitterError(err error) error {
	logger.Warn("emitter error", zap.Error(err))
	return nil
}

func (e *ensEngine) createPool(pool *entity.Uniswapv3PoolCreated) error {
	logger.Debug("createPool: got pool", zap.Any("pool", pool))

	return nil
}
