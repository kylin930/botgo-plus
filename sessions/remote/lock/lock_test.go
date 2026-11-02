package lock

import (
	"context"
	"os"
	"testing"
	"time"

	redis "github.com/go-redis/redis/v8"
)

func TestLock_Lock(t *testing.T) {
	address := os.Getenv("BOTGO_PLUS_REDIS_TEST_ADDR")
	if address == "" {
		t.Skip("Redis integration not executed; set BOTGO_PLUS_REDIS_TEST_ADDR explicitly")
	}
	conn := redis.NewClient(&redis.Options{
		Addr:         address,
		DialTimeout:  800 * time.Millisecond,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	defer conn.Close()
	key := "botgo-plus-test-" + time.Now().Format("20060102150405.000000000")
	value := "ddd"
	expire := 4 * time.Second
	ctx := context.Background()

	lock := New(key, value, conn)

	t.Run("lock", func(t *testing.T) {
		if err := lock.Lock(ctx, expire); err != nil {
			t.Error(err)
		}
	})
	t.Run("renew", func(t *testing.T) {
		if err := lock.Renew(ctx, expire); err != nil {
			t.Error(err)
		}
	})
	t.Run("release", func(t *testing.T) {
		if err := lock.Release(ctx); err != nil {
			t.Error(err)
		}
	})

	t.Run("renew goroutine and check", func(t *testing.T) {
		if err := lock.Lock(ctx, expire); err != nil {
			t.Error(err)
		}
		go lock.StartRenew(ctx, expire)
		time.Sleep(expire + 2*time.Second)
		// renew 持续在跑，这里不应该再抢到锁
		if err := lock.Lock(ctx, expire); err == nil {
			t.Error("want lock err, but got nil")
		}
		lock.StopRenew()
		time.Sleep(expire + 2*time.Second)
		if err := lock.Lock(ctx, expire); err != nil {
			t.Error(err)
		}
	})
}
