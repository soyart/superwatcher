package entity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

func redisEnsKey(key string) string {
	return fmt.Sprintf("demo:ens:%s", key)
}

type EnsDataGateway struct {
	redisClient redis.Client
}

func NewEnsDataGateway(
	redisCli redis.Client,
) *EnsDataGateway {
	return &EnsDataGateway{
		redisClient: redisCli,
	}
}

// handleRedisErr checks if err is redis.Nil
func handleRedisErr(err error, action, key string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, redis.Nil) {
		err = errors.Wrap(ErrRecordNotFound, err.Error())
		err = WrapErrRecordNotFound(err, key)
	}
	return errors.Wrapf(err, "action: %s", action)
}

func (s *EnsDataGateway) SaveENS(
	ctx context.Context,
	ens *ENS,
) error {
	key := redisEnsKey(ens.ID)
	return handleRedisErr(
		s.redisClient.Set(ctx, key, ens, -1).Err(),
		"set RecordedENS",
		key,
	)
}

func (s *EnsDataGateway) GetRecordedENS(
	ctx context.Context,
	key string,
) (*ENS, error) {
	stringData, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, handleRedisErr(err, "get RecordedENS", key)
	}

	ens := &ENS{}
	err = json.Unmarshal([]byte(stringData), ens)
	if err != nil {
		return nil, errors.Wrapf(err, "action: %s", "unmarshal RecordedENS")
	}
	return ens, nil
}

func (s *EnsDataGateway) GetRecordedENSs(
	ctx context.Context,
) ([]*ENS, error) {
	ENSs := []*ENS{}
	var cursor uint64
	prefix := redisEnsKey("*")

	for {
		var keys []string
		var err error
		keys, cursor, err = s.redisClient.Scan(ctx, cursor, prefix, 0).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "action: %s", "scan RecordedENS")
		}

		for _, key := range keys {
			ens, err := s.GetRecordedENS(ctx, key)
			if err != nil {
				return nil, errors.Wrapf(err, "action: %s", "get RecordedENS")
			}

			ENSs = append(ENSs, ens)
		}

		if cursor == 0 {
			break
		}
	}

	return ENSs, nil
}

func (s *EnsDataGateway) RemoveENS(
	ctx context.Context,
	key string,
) error {
	_, err := s.redisClient.Del(ctx, key).Result()
	if err != nil {
		return handleRedisErr(err, "del RecordedENS", key)
	}
	return nil
}

func (s *EnsDataGateway) Shutdown() error {
	return s.redisClient.Close()
}
