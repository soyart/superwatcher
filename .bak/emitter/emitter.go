package emitter

// This package maybe removed in favor of centralized pkg/components

import (
	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/internal/emitter"
)

type config struct {
	conf                *superwatcher.Config
	poller              superwatcher.EmitterPoller
	ethClient           superwatcher.EthClient
	getStateDataGateway superwatcher.GetStateDataGateway
	syncChan            <-chan struct{}
	pollResultChan      chan<- *superwatcher.PollerResult
	errChan             chan<- error
}

type Option func(*config)

func WithConfig(conf *superwatcher.Config) Option {
	return func(c *config) {
		c.conf = conf
	}
}

func WithEmitterPoller(poller superwatcher.EmitterPoller) Option { //nolint:revive
	return func(c *config) {
		c.poller = poller
	}
}

func WithEthClient(client superwatcher.EthClient) Option {
	return func(c *config) {
		c.ethClient = client
	}
}

func WithGetStateDataGateway(gateway superwatcher.StateDataGateway) Option {
	return func(c *config) {
		c.getStateDataGateway = gateway
	}
}

func WithSyncChan(syncChan <-chan struct{}) Option {
	return func(c *config) {
		c.syncChan = syncChan
	}
}

func WithFilterResultChan(resultChan chan<- *superwatcher.PollerResult) Option {
	return func(c *config) {
		c.pollResultChan = resultChan
	}
}

func WithErrChan(errChan chan<- error) Option {
	return func(c *config) {
		c.errChan = errChan
	}
}

func New(options ...Option) superwatcher.Emitter {
	var c config
	for _, opt := range options {
		opt(&c)
	}

	return emitter.New(
		c.conf,
		c.ethClient,
		c.getStateDataGateway,
		c.poller,
		c.syncChan,
		c.pollResultChan,
		c.errChan,
	)
}
