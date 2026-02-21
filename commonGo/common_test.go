package commonGo

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCronJob(t *testing.T) {
	t.Parallel()

	t.Run("should work", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		counter := uint64(0)
		handler := func(ctx context.Context) {
			atomic.AddUint64(&counter, 1)
		}

		CronJobStarter(ctx, handler, time.Millisecond*100)

		time.Sleep(time.Millisecond * 350) // 350ms => 3 calls => counter should be 3

		assert.Equal(t, uint64(3), atomic.LoadUint64(&counter))
	})
	t.Run("context done should stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		counter := uint64(0)
		handler := func(ctx context.Context) {
			atomic.AddUint64(&counter, 1)
		}

		CronJobStarter(ctx, handler, time.Millisecond*100)

		time.Sleep(time.Millisecond * 350) // 35oms => 3 calls => counter should be 3
		cancel()

		time.Sleep(time.Millisecond * 350) // wait another 350ms just to be safe

		assert.Equal(t, uint64(3), atomic.LoadUint64(&counter))
	})
}
