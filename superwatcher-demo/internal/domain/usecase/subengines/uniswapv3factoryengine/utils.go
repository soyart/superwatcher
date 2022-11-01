package uniswapv3factoryengine

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/artnoi43/superwatcher/pkg/logger"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/entity"
	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/lib/logutils"
)

func (e *uniswapv3PoolFactoryEngine) handlePoolCreated(
	pool *entity.Uniswapv3PoolCreated,
) error {
	logger.Info("got poolCreated, writing to db", zap.Any("pool", pool))

	return nil
}

func (e *uniswapv3PoolFactoryEngine) revertPoolCreated(
	pool *entity.Uniswapv3PoolCreated,
) error {
	logger.Info("reverting poolCreated", zap.Any("pool", pool))

	return nil
}

// parsePoolCreatedUnpackedMap collects unpacked log.Data into *entity.Uniswapv3PoolCreated.
// Other fields not available in the log byte data is populated elsewhere.
func parsePoolCreatedUnpackedMap(unpacked map[string]interface{}) (*entity.Uniswapv3PoolCreated, error) {
	poolAddr, err := logutils.ExtractFieldFromUnpacked[common.Address](unpacked, "pool")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unpack poolAddr from map %v", unpacked)
	}
	return &entity.Uniswapv3PoolCreated{
		Address: poolAddr,
	}, nil
}