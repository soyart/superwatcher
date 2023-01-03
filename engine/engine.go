package engine

import (
	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/internal/engine"
)

type config struct {
	emitterClient    superwatcher.EmitterClient
	serviceEngine    superwatcher.ServiceEngine
	stateDataGateway superwatcher.SetStateDataGateway
	logLevel         uint8
}

type Option func(*config)

func WithEmitterClient(client superwatcher.EmitterClient) Option {
	return func(c *config) {
		c.emitterClient = client
	}
}

func WithServiceEngine(service superwatcher.ServiceEngine) Option {
	return func(c *config) {
		c.serviceEngine = service
	}
}

func WithSetStateDataGateway(gateway superwatcher.SetStateDataGateway) Option {
	return func(c *config) {
		c.stateDataGateway = gateway
	}
}

func WithLogLevel(level uint8) Option {
	return func(c *config) {
		c.logLevel = level
	}
}

func New(options ...Option) superwatcher.Engine {
	var c config
	for _, opt := range options {
		opt(&c)
	}

	return engine.New(
		c.emitterClient,
		c.serviceEngine,
		c.stateDataGateway,
		c.logLevel,
	)
}
