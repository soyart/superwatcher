package engine

import (
	"context"

	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher/domain/datagateway"
	"github.com/artnoi43/superwatcher/domain/usecase/emitterclient"
	"github.com/artnoi43/superwatcher/lib/logger/debug"
)

type WatcherEngine interface {
	Loop(context.Context) error
}

type engine struct {
	emitterClient    emitterclient.Client         // Interfaces with emitter
	stateDataGateway datagateway.StateDataGateway // Saves lastRecordedBlock to Redis
	metadataTracker  MetadataTracker              // Engine internal state machine

	serviceEngine ServiceEngine // Injected service code

	debug bool
}

func (e *engine) Loop(ctx context.Context) error {
	go func() {
		if err := e.handleResult(ctx); err != nil {
			e.debugMsg("*engine.run exited", zap.Error(err))
		}

		defer e.shutdown()
	}()

	return e.handleEmitterError()
}

// shutdown is not exported, and the user of the engine should not attempt to call it.
func (e *engine) shutdown() {
	e.stateDataGateway.Shutdown()
	e.emitterClient.Shutdown()
}
func (e *engine) debugMsg(msg string, fields ...zap.Field) {
	debug.DebugMsg(e.debug, msg, fields...)
}
