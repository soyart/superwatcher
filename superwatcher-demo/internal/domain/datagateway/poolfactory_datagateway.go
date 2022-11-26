package datagateway

import (
	"context"
	"encoding/json"

	"github.com/artnoi43/gsl/gslutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"

	"github.com/artnoi43/superwatcher/superwatcher-demo/internal/domain/entity"
)

const PoolFactoryRedisKey = "demo:poolfactory"

type DataGatewayPoolFactory interface {
	SetPool(context.Context, *entity.Uniswapv3PoolCreated) error
	GetPool(context.Context, common.Address) (*entity.Uniswapv3PoolCreated, error)
	GetPools(context.Context) ([]*entity.Uniswapv3PoolCreated, error)
	DelPool(context.Context, *entity.Uniswapv3PoolCreated) error
}

type dataGatewayPoolFactory struct {
	redisCli *redis.Client
}

func NewDataGatewayPoolFactory(redisCli *redis.Client) DataGatewayPoolFactory {
	return &dataGatewayPoolFactory{
		redisCli: redisCli,
	}
}

func (s *dataGatewayPoolFactory) SetPool(
	ctx context.Context,
	pool *entity.Uniswapv3PoolCreated,
) error {
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal pool %s", pool.Address.String())
	}

	addr := gslutils.StringerToLowerString(pool.Address)
	if err := s.redisCli.HSet(ctx, PoolFactoryRedisKey, addr, poolJSON).Err(); err != nil {
		return handleRedisErr(err, "HSet pool", addr)
	}

	return nil
}

func (s *dataGatewayPoolFactory) GetPool(
	ctx context.Context,
	lpAddress common.Address,
) (
	*entity.Uniswapv3PoolCreated,
	error,
) {
	addr := gslutils.StringerToLowerString(lpAddress)
	poolJSON, err := s.redisCli.HGet(ctx, PoolFactoryRedisKey, addr).Result()
	if err != nil {
		return nil, handleRedisErr(err, "HGET pool", addr)
	}

	var pool entity.Uniswapv3PoolCreated
	if err := json.Unmarshal([]byte(poolJSON), &pool); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal poolJSON")
	}

	return &pool, nil
}

func (s *dataGatewayPoolFactory) GetPools(
	ctx context.Context,
) (
	[]*entity.Uniswapv3PoolCreated,
	error,
) {
	resultMap, err := s.redisCli.HGetAll(ctx, PoolFactoryRedisKey).Result()
	if err != nil {
		return nil, handleRedisErr(err, "HGETALL pool", "null")
	}

	var pools []*entity.Uniswapv3PoolCreated
	for lpAddress, poolJSON := range resultMap {
		var pool entity.Uniswapv3PoolCreated
		if err := json.Unmarshal([]byte(poolJSON), &pool); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal poolJSON %s", lpAddress)
		}

		pools = append(pools, &pool)
	}

	return pools, nil
}

func (s *dataGatewayPoolFactory) DelPool(
	ctx context.Context,
	pool *entity.Uniswapv3PoolCreated,
) error {
	addr := gslutils.StringerToLowerString(pool.Address)
	if err := s.redisCli.HDel(ctx, PoolFactoryRedisKey, addr).Err(); err != nil {
		return handleRedisErr(err, "HDEL pool", addr)
	}

	return nil
}
