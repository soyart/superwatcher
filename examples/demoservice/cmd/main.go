package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/soyart/gsl/soyutils"
	"github.com/soyart/w3utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	// Most application/service code should only import these superwatcher packages, not `internal`.
	"github.com/soyart/superwatcher"
	"github.com/soyart/superwatcher/pkg/components"
	"github.com/soyart/superwatcher/pkg/logger"

	"github.com/soyart/superwatcher/examples/demoservice/config"
	"github.com/soyart/superwatcher/examples/demoservice/internal/domain/datagateway"
	"github.com/soyart/superwatcher/examples/demoservice/internal/domain/datagateway/watcherstate"
	"github.com/soyart/superwatcher/examples/demoservice/internal/hardcode"
	"github.com/soyart/superwatcher/examples/demoservice/internal/routerengine"
	"github.com/soyart/superwatcher/examples/demoservice/internal/subengines"
	"github.com/soyart/superwatcher/examples/demoservice/internal/subengines/ensengine"
	"github.com/soyart/superwatcher/examples/demoservice/internal/subengines/uniswapv3factoryengine"
)

func main() {
	conf, err := soyutils.ReadFileYAMLPointer[config.Config]("./examples/demoservice/config/config.yaml")
	if err != nil {
		panic("failed to read YAML config: " + err.Error())
	}

	chain := conf.Chain
	if chain == "" {
		panic("empty chain")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: conf.RedisConnAddr,
	})
	if rdb == nil {
		panic("nil redis")
	}

	stateDataGateway, err := watcherstate.NewRedisStateDataGateway(
		"demoservice"+":"+chain,
		rdb,
	)
	if err != nil {
		panic("new stateDataGateway failed: " + err.Error())
	}

	// Hard-coded topic values for testing
	demoContracts := hardcode.DemoContracts(
		hardcode.Uniswapv3Factory,
		hardcode.ENSRegistrar,
		hardcode.ENSController,
	)

	// Init demo service instances and items with demoContracts
	emitterAddresses, emitterTopics, demoRoutes, demoServices := contractsToServices(demoContracts, rdb, conf.SuperWatcherConfig.LogLevel)
	logger.Debug("init: addresses", zap.Any("emitterAddresses", emitterAddresses))
	logger.Debug("init: topics", zap.Any("emitterTopics", emitterTopics))
	logger.Debug("init: demoRoutes", zap.Any("demoRoutes", demoRoutes))
	logger.Debug("init: demoServices", zap.Any("demoServices", demoServices))

	// It will later wraps uniswapv3PoolEngine and oneInchLimitOrderEngine
	// and like wise needs their FSMs too.
	demoEngine := routerengine.New(
		demoRoutes,
		demoServices,
		conf.SuperWatcherConfig.LogLevel,
	)

	syncChan := make(chan struct{})
	pollResultChan := make(chan *superwatcher.PollerResult)
	errChan := make(chan error)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	// There are many ways to init superwatcher components. See package pkg/components
	watcher := components.NewSuperWatcherOptions(
		components.WithConfig(conf.SuperWatcherConfig),
		components.WithEthClient(superwatcher.NewEthClient(ctx, conf.SuperWatcherConfig.NodeURL)),
		components.WithGetStateDataGateway(stateDataGateway),
		components.WithSetStateDataGateway(stateDataGateway),
		components.WithServiceEngine(demoEngine),
		components.WithSyncChan(syncChan),
		components.WithFilterResultChan(pollResultChan),
		components.WithErrChan(errChan),
		components.WithAddresses(emitterAddresses...),
		components.WithTopics(emitterTopics),
	)

	if err := watcher.Run(ctx, cancel); err != nil {
		logger.Debug("watcher.Run exited", zap.Error(err))
	}

	// =====================================================================

	// Or you can use a more direct approach without using options
	// watcher := components.NewSuperWatcherDefault(
	// 	conf.SuperWatcherConfig,
	// 	superwatcher.WrapEthClient(ethClient),
	// 	// We wrap |stateDataGateway| to demo how to separate the 2 methods
	// 	// for a superwatcher.StateDataGateway for single responsibility.
	// 	superwatcher.GetStateDataGatewayFunc(stateDataGateway.GetLastRecordedBlock),
	// 	superwatcher.SetStateDataGatewayFunc(stateDataGateway.SetLastRecordedBlock),
	// 	demoEngine,
	// 	emitterAddresses,
	// 	[][]common.Hash{emitterTopics},
	// )

	// =====================================================================

	// Alternatively, we can run the components manually
	// watcherEmitter, watcherEngine := newSuperWatcherPreferred(
	// 	conf.SuperWatcherConfig,
	// 	ethClient,
	// 	emitterAddresses,
	// 	emitterTopics,
	// 	stateDataGateway,
	// 	demoEngine,
	// )
	//
	// // Graceful shutdown
	// defer func() {
	// 	// Cancel context to stop both superwatcher emitter and engine
	// 	cancel()
	//
	// 	ethClient.Close()
	// 	if err := rdb.Close(); err != nil {
	// 		logger.Error(
	// 			"error during graceful shutdown - Redis client not properly closed",
	// 			zap.Error(err),
	// 		)
	// 	}
	//
	// 	logger.Info("graceful shutdown successful")
	// }()
	//
	// go func() {
	// 	if err := watcherEmitter.Loop(ctx); err != nil {
	// 		logger.Error("DEMO: emitter returned an error", zap.Error(err))
	// 	}
	// }()
	//
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	if err := watcherEngine.Loop(ctx); err != nil {
	// 		logger.Error("DEMO: engine returned an error", zap.Error(err))
	// 	}
	// }()
	//
	// // Demo how to use SetDoReorg
	// go func() {
	// 	time.Sleep(5 * time.Second)
	// 	watcherEmitter.Poller().SetDoReorg(false)
	// }()
	//
	// wg.Wait()
}

func contractsToServices(
	demoContracts map[string]w3utils.Contract,
	rdb *redis.Client,
	logLevel uint8,
) (
	[]common.Address,
	[]common.Hash,
	map[subengines.SubEngineEnum]map[common.Address][]common.Hash, // demoRoutes
	map[subengines.SubEngineEnum]superwatcher.ServiceEngine, // demoServices
) {
	// Demo sub-engines
	demoRoutes := make(map[subengines.SubEngineEnum]map[common.Address][]common.Hash)
	demoServices := make(map[subengines.SubEngineEnum]superwatcher.ServiceEngine)

	dgwENS := datagateway.NewEnsDataGateway(rdb)
	dgwPoolFactory := datagateway.NewDataGatewayPoolFactory(rdb)

	// ENS sub-engine has 2 contracts
	// so we can't init the engine in the for loop below
	var ensRegistrar, ensController w3utils.Contract
	// Topics and addresses to be used by watcher emitter
	var emitterTopics []common.Hash
	var emitterAddresses []common.Address //nolint:prealloc
	for contractName, demoContract := range demoContracts {
		contractTopics := make([]common.Hash, len(demoContract.ContractEvents))
		var subEngine subengines.SubEngineEnum

		switch contractName {
		case hardcode.Uniswapv3Factory:
			subEngine = subengines.SubEngineUniswapv3Factory
			demoServices[subEngine] = uniswapv3factoryengine.New(demoContract, dgwPoolFactory, logLevel)

		case hardcode.ENSRegistrar, hardcode.ENSController:
			// demoServices for ENS will be created outside of this for loop
			subEngine = subengines.SubEngineENS
			if contractName == hardcode.ENSRegistrar {
				ensRegistrar = demoContract
			} else {
				ensController = demoContract
			}
		}

		for i, event := range demoContract.ContractEvents {
			contractTopics[i] = event.ID
		}

		if demoRoutes[subEngine] == nil {
			demoRoutes[subEngine] = make(map[common.Address][]common.Hash)
		}
		demoRoutes[subEngine][demoContract.Address] = contractTopics
		emitterAddresses = append(emitterAddresses, demoContract.Address)
	}

	// Initialize ensEngine
	ensEngine := ensengine.New(ensRegistrar, ensController, dgwENS, logLevel)
	demoServices[subengines.SubEngineENS] = ensEngine

	return emitterAddresses, emitterTopics, demoRoutes, demoServices
}
