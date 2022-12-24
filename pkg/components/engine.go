package components

import (
	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/config"
	"github.com/artnoi43/superwatcher/internal/engine"
)

func NewEngine(
	emitterClient superwatcher.EmitterClient,
	serviceEngine superwatcher.ServiceEngine,
	stateDataGateway superwatcher.SetStateDataGateway,
	logLevel uint8,
) superwatcher.Engine {
	return engine.New(
		emitterClient,
		serviceEngine,
		stateDataGateway,
		logLevel,
	)
}

// NewEngineWithEmitterClient creates a new superwatcher.Engine, and pair it with an superwatcher.EmitterClient.
// This is the preferred way of creating a new superwatcher.Engine
func NewEngineWithEmitterClient(
	conf *config.Config,
	serviceEngine superwatcher.ServiceEngine,
	stateDataGateway superwatcher.SetStateDataGateway,
	syncChan chan<- struct{},
	filterResultChan <-chan *superwatcher.FilterResult,
	errChan <-chan error,
) superwatcher.Engine {
	// TODO: Do we still need EmitterClient?
	emitterClient := NewEmitterClient(
		conf,
		syncChan,
		filterResultChan,
		errChan,
	)

	return NewEngine(
		emitterClient,
		serviceEngine,
		stateDataGateway,
		conf.LogLevel,
	)
}