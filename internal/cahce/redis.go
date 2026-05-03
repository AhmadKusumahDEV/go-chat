package cahce

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CahceRedis interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) error
	Del(ctx context.Context, key string) error
	FlushAll(ctx context.Context) error
}

type clientRoomService struct {
	client *redis.Client
}

// Del implements CahceRedis.
func (c *clientRoomService) Del(ctx context.Context, key string) error {
	ctx, cancle := context.WithTimeout(ctx, 3*time.Second)
	defer cancle()

	return c.client.Del(ctx, key).Err()
}

// FlushAll implements CahceRedis.
func (c *clientRoomService) FlushAll(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

// Get implements CahceRedis.
func (c *clientRoomService) Get(ctx context.Context, key string, dest any) error {
	result, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return errors.New("miss cahce")
	} else if err != nil {
		return err
	}

	return json.Unmarshal([]byte(result), dest)
}

// Set implements CahceRedis.
func (c *clientRoomService) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	ctx, cancle := context.WithTimeout(ctx, 3*time.Second)
	defer cancle()

	jsonValue, err := json.Marshal(value)
	if err != nil {
		log.Println("error on json marshal redis layer", err)
		return err
	}

	err = c.client.Set(ctx, key, jsonValue, ttl).Err()
	if err != nil {
		log.Println("error on set redis layer", err)
		return err
	}

	log.Println("successfuly set redis with key ", key)

	return nil
}

func NewClientRedis(client *redis.Client) CahceRedis {
	return &clientRoomService{
		client: client,
	}
}
