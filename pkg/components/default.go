package components

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/config"
)

// NewDefault returns default implementations of WatcherEmitter and WatcherEngine.
// The EmitterClient is initialized and embedded to the returned engine within this function.
// This is the preferred way for initializing superwatcher components.
func NewDefault(
	conf *config.Config,
	ethClient superwatcher.EthClient,
	getStateDataGateway superwatcher.GetStateDataGateway,
	setStateDataGateway superwatcher.SetStateDataGateway,
	addresses []common.Address,
	topics [][]common.Hash,
	serviceEngine superwatcher.ServiceEngine,
) (
	superwatcher.Emitter,
	superwatcher.Engine,
) {
	syncChan := make(chan struct{})
	filterResultChan := make(chan *superwatcher.FilterResult)
	errChan := make(chan error)

	watcherEmitter := NewEmitterWithPoller(
		conf,
		ethClient,
		getStateDataGateway,
		addresses,
		topics,
		syncChan,
		filterResultChan,
		errChan,
	)

	watcherEngine := NewEngineWithEmitterClient(
		conf,
		serviceEngine,
		setStateDataGateway,
		syncChan,
		filterResultChan,
		errChan,
	)

	return watcherEmitter, watcherEngine
}