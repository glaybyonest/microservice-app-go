package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func InitRedis(addr string) error {
	Rdb = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// GetFromCache получает данные из кэша. Возвращает true, если найдено.
func GetFromCache(key string, dest interface{}) (bool, error) {
	val, err := Rdb.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

// SetToCache сохраняет данные в кэш с TTL.
func SetToCache(key string, data interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return Rdb.Set(context.Background(), key, jsonData, ttl).Err()
}

// InvalidateCache удаляет ключи из кэша.
func InvalidateCache(keys ...string) error {
	return Rdb.Del(context.Background(), keys...).Err()
}
