package components

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/soyart/gsl"

	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/pkg/logger/debugger"
)

// superWatcher implements superwatcher.SuperWatcher.
// It is a meta-type in that it merely wraps Emitter and Engine,
// and only provides superWatcher.Run as its original method.
type superWatcher struct {
	emitter  superwatcher.Emitter
	engine   superwatcher.Engine
	debugger *debugger.Debugger
}

func NewSuperWatcherOptions(options ...Option) superwatcher.SuperWatcher {
	var conf componentConfig
	for _, opt := range options {
		opt(&conf)
	}

	logLevel := gsl.Max(conf.logLevel, conf.config.LogLevel)

	poller := NewPoller(
		conf.addresses,
		conf.topics,
		conf.config.DoReorg || conf.doReorg,
		conf.config.DoHeader || conf.doHeader,
		conf.filterRange,
		conf.ethClient,
		logLevel,
		gsl.Max(conf.policy, conf.config.Policy),
	)

	emitter := NewEmitter(
		conf.config,
		conf.ethClient,
		conf.getStateDataGateway,
		poller,
		conf.syncChan,
		conf.pollResultChan,
		conf.errChan,
	)

	emitterClient := NewEmitterClient(
		conf.config,
		conf.syncChan,
		conf.pollResultChan,
		conf.errChan,
	)

	engine := NewEngine(
		emitterClient,
		conf.serviceEngine,
		conf.setStateDataGateway,
		logLevel,
	)

	return NewSuperWatcher(emitter, engine, logLevel)
}

func NewSuperWatcherDefault(
	conf *superwatcher.Config,
	ethClient superwatcher.EthClient,
	getStateDataGateway superwatcher.GetStateDataGateway,
	setStateDataGateway superwatcher.SetStateDataGateway,
	serviceEngine superwatcher.ServiceEngine,
	addresses []common.Address,
	topics [][]common.Hash,
) superwatcher.SuperWatcher {
	emitter, engine := NewDefault(
		conf,
		ethClient,
		getStateDataGateway,
		setStateDataGateway,
		serviceEngine,
		addresses,
		topics,
	)

	return NewSuperWatcher(emitter, engine, conf.LogLevel)
}

func NewSuperWatcher(
	emitter superwatcher.Emitter,
	engine superwatcher.Engine,
	logLevel uint8,
) superwatcher.SuperWatcher {
	return &superWatcher{
		emitter:  emitter,
		engine:   engine,
		debugger: debugger.NewDebugger("SuperWatcher", logLevel),
	}
}
