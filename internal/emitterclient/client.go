package emitterclient

import (
	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/config"
	"github.com/artnoi43/superwatcher/pkg/logger/debugger"
)

// emitterClient is the actual implementation of EmitterClient.
// It uses channels to communicate with emitter.
type emitterClient struct {
	emitterConfig   *config.Config
	emitterSyncChan chan<- struct{}
	pollResultChan  <-chan *superwatcher.PollResult
	errChan         <-chan error

	debugger *debugger.Debugger
}

func New(
	emitterConfig *config.Config,
	emitterSyncChan chan<- struct{},
	pollResultChan <-chan *superwatcher.PollResult,
	errChan <-chan error,
	logLevel uint8,
) superwatcher.EmitterClient {
	return &emitterClient{
		emitterConfig:   emitterConfig,
		pollResultChan:  pollResultChan,
		emitterSyncChan: emitterSyncChan,
		errChan:         errChan,
		debugger:        debugger.NewDebugger("emitter-client", logLevel),
	}
}

func (c *emitterClient) Shutdown() {
	c.debugger.Debug(2, "emitterClient.Shutdown() called")

	if c.emitterSyncChan != nil {
		c.debugger.Debug(2, "closing emitterClient.emitterSyncChan")
		close(c.emitterSyncChan)
	} else {
		c.debugger.Debug(2, "emitterSyncChan was already closed")
	}
}

func (c *emitterClient) SyncsEmitter() {
	c.emitterSyncChan <- struct{}{}
}

func (c *emitterClient) WatcherConfig() *config.Config {
	return c.emitterConfig
}

func (c *emitterClient) WatcherResult() *superwatcher.PollResult {
	result, ok := <-c.pollResultChan
	if ok {
		return result
	}

	c.debugger.Debug(2, "pollResultChan was closed")
	return nil
}

func (c *emitterClient) WatcherError() error {
	err, ok := <-c.errChan
	if ok {
		return err
	}

	c.debugger.Debug(2, "errChan was closed")
	return nil
}
