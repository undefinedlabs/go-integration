package example

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-redis/redis"

	"github.com/undefinedlabs/go-integration"
)

var testData = map[string]string{
	"testdata": "this should be available in all tests",
}

func setupRedis(svc *integration.Service) error {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:6379", svc.Hostname()), MaxRetries: 10})
	defer client.Close()
	for k, v := range testData {
		_, err := client.Set(k, v, 0).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

var redisSvc = integration.NewService("redis", "redis:latest", integration.WithSetup(setupRedis))

func TestIntegration(t *testing.T) {
	it := integration.NewIntegrationTest(t, integration.DependsOn(redisSvc))
	it.Run(func(ctx context.Context, t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:6379", redisSvc.Hostname())})
		defer client.Close()
		pong, err := client.Ping().Result()
		if err != nil {
			t.Fatalf("ping failed: %v", err)
		}
		t.Logf("ping: %s", pong)
	})
}
