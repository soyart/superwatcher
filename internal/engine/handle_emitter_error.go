package engine

import "github.com/pkg/errors"

func (e *engine) handleEmitterError() error {
	e.debugMsg("*engine.handleError started")
	for {
		err := e.emitterClient.WatcherError()
		if err != nil {
			err = e.serviceEngine.HandleEmitterError(err)
			if err != nil {
				return errors.Wrap(err, "serviceEngine failed to handle error")
			}

			// Emitter error handled in service without error
			continue
		}

		e.debugMsg("got nil error from emitter - should not happen unless errChan was closed")
		break
	}
	return nil
}