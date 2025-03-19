package bloomfilter

import (
	"context"
	"fmt"
	"github.com/dapplink-labs/multichain-sync-btc/config"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"strconv"
)

type BloomFilter struct {
	rdb *redis.Client
}

func InitBloomFilter(ctx context.Context, config *config.RedisConfig, db *database.DB) (*BloomFilter, error) {
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: config.Password,
		DB:       config.DB,
	})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Error("failed to connect to redis", "err", err)
		return nil, err
	}
	bfs := &BloomFilter{
		rdb: client,
	}
	businessList, err := db.Business.QueryBusinessList()
	for _, biz := range businessList {
		addresses, _ := db.Addresses.GetAllAddresses(biz.BusinessUid)
		length := len(addresses)
		if length > 0 {
			result, _ := client.Exists(ctx, biz.BusinessUid).Result()
			if result != 1 {
				i, _ := client.Do(ctx, "BF.RESERVE", biz.BusinessUid, "0.01", strconv.Itoa(length)).Result()
				if i != 1 {
					log.Error("create bloom filter failed")
					continue
				}
			}
			for _, addr := range addresses {
				key := fmt.Sprintf(addr.Address, addr.AddressType)
				exists, _ := client.Do(ctx, "BF.EXISTS", biz.BusinessUid, key).Int()
				if exists != 0 {
					client.Do(ctx, "BF.ADD", biz.BusinessUid, key)
				}
			}
		}
	}
	return bfs, nil
}

func (bf *BloomFilter) Add(ctx context.Context, requestId string, item string, addressType int) error {
	e, _ := bf.rdb.Exists(ctx, requestId).Result()
	if e != 1 {
		i, _ := bf.rdb.Do(ctx, "BF.RESERVE", requestId, 0.01, 100).Result()
		if i != 1 {
			log.Error("create bloom filter failed")
		}
	}
	key := fmt.Sprintf(item, addressType)
	_, err := bf.rdb.Do(ctx, "BF.ADD", requestId, key).Result()
	return err
}

func (bf *BloomFilter) Exists(ctx context.Context, requestId string, key string) (int, error) {
	i, _ := bf.rdb.Exists(ctx, requestId).Result()
	if i != 1 {
		return -1, nil
	}
	userKey := fmt.Sprintf(key, "0")
	result, _ := bf.rdb.Do(ctx, "BF.EXISTS", requestId, userKey).Int()
	if result == 1 {
		return 0, nil
	}
	hotKey := fmt.Sprintf(key, "1")
	result, _ = bf.rdb.Do(ctx, "BF.EXISTS", requestId, hotKey).Int()
	if result == 1 {
		return 1, nil
	}
	coldKey := fmt.Sprintf(key, "2")
	result, _ = bf.rdb.Do(ctx, "BF.EXISTS", requestId, coldKey).Int()
	if result == 1 {
		return 2, nil
	}
	return -1, nil
}
