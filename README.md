# go-integration

**NOTE: this is only a POC. Use only if you know what you are doing**

Integration test package for [Yoonit](https://yoonit.io)


## Usage

Example of an integration test with a Redis service:

```go
package myapp

import (
	"testing"
	"github.com/yoonitio/go-integration"
	"github.com/go-redis/redis"
	"fmt"
)

var testData = map[string]string{
	"testdata": "this should be available in all tests",
}

func setupRedis(svc *integration.Service) error {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:6379", svc.Hostname())})
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
	it.Run("test", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:6379", redisSvc.Hostname())})
		defer client.Close()
		pong, err := client.Ping().Result()
		if err != nil {
			t.Fatalf("ping failed: %v", err)
		}
		t.Logf("ping: %s", pong)
	})
}
```