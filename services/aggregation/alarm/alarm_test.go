package alarm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/testsCommon"
	"github.com/stretchr/testify/assert"
)

func TestNewAlarmService(t *testing.T) {
	t.Parallel()

	t.Run("nil store should error", func(t *testing.T) {
		alarm, err := NewAlarmService(
			nil,
			&testsCommon.OutputNotifiersHandlerStub{},
			&testsCommon.StatusHandlerStub{},
			1,
			time.Second)

		assert.Nil(t, alarm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil storage provided to alarm service")
		assert.True(t, alarm.IsInterfaceNil())
	})
	t.Run("nil output notifiers handler should error", func(t *testing.T) {
		alarm, err := NewAlarmService(
			&testsCommon.StoreStub{},
			nil,
			&testsCommon.StatusHandlerStub{},
			1,
			time.Second)

		assert.Nil(t, alarm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil output notifiers handler provided to alarm service")
		assert.True(t, alarm.IsInterfaceNil())
	})
	t.Run("nil status handler should error", func(t *testing.T) {
		alarm, err := NewAlarmService(
			&testsCommon.StoreStub{},
			&testsCommon.OutputNotifiersHandlerStub{},
			nil,
			1,
			time.Second)

		assert.Nil(t, alarm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil status handler provided to alarm service")
		assert.True(t, alarm.IsInterfaceNil())
	})
	t.Run("invalid num seconds to consider stale should error", func(t *testing.T) {
		alarm, err := NewAlarmService(
			&testsCommon.StoreStub{},
			&testsCommon.OutputNotifiersHandlerStub{},
			&testsCommon.StatusHandlerStub{},
			0,
			time.Second)

		assert.Nil(t, alarm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "num seconds to consider stale must be greater than 0")
		assert.True(t, alarm.IsInterfaceNil())
	})
	t.Run("invalid loop duration should error", func(t *testing.T) {
		alarm, err := NewAlarmService(
			&testsCommon.StoreStub{},
			&testsCommon.OutputNotifiersHandlerStub{},
			&testsCommon.StatusHandlerStub{},
			1,
			time.Millisecond)

		assert.Nil(t, alarm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "loop time must be greater than 10ms")
		assert.True(t, alarm.IsInterfaceNil())
	})
	t.Run("should work", func(t *testing.T) {
		alarm, err := NewAlarmService(
			&testsCommon.StoreStub{},
			&testsCommon.OutputNotifiersHandlerStub{},
			&testsCommon.StatusHandlerStub{},
			1,
			time.Millisecond*10)

		assert.NotNil(t, alarm)
		assert.Nil(t, err)
		assert.False(t, alarm.IsInterfaceNil())
	})
}

func TestAlarmService_StartClose(t *testing.T) {
	t.Parallel()

	numCalls := uint32(0)
	alarm, _ := NewAlarmService(
		&testsCommon.StoreStub{
			GetLatestMetricsHandler: func(ctx context.Context) ([]common.MetricHistory, error) {
				atomic.AddUint32(&numCalls, 1)
				return make([]common.MetricHistory, 0), nil
			},
		},
		&testsCommon.OutputNotifiersHandlerStub{},
		&testsCommon.StatusHandlerStub{},
		1,
		time.Millisecond*100)

	time.Sleep(time.Second)
	// numCalls should be 0 as we did not start the loop
	assert.Equal(t, uint32(0), atomic.LoadUint32(&numCalls))

	alarm.Start()
	alarm.Start() // nothing happens as we already started the loop
	time.Sleep(time.Millisecond * 350)

	// 3 calls
	assert.Equal(t, uint32(3), atomic.LoadUint32(&numCalls))

	err := alarm.Close()
	assert.Nil(t, err)

	err = alarm.Close() // nothing happens as we already stopped the loop
	assert.Nil(t, err)

	time.Sleep(time.Second)
	// numCalls should still be 3 as we stopped the loop
	assert.Equal(t, uint32(3), atomic.LoadUint32(&numCalls))
}

func TestAlarmService_Notifications(t *testing.T) {
	t.Parallel()

	t.Run("one metric without history should notify once", func(t *testing.T) {
		t.Parallel()

		notifyWithRetryNumCalled := uint32(0)
		collectKeysProblemsNumCalled := uint32(0)

		alarm, _ := NewAlarmService(
			&testsCommon.StoreStub{
				GetLatestMetricsHandler: func(ctx context.Context) ([]common.MetricHistory, error) {
					metric1 := common.MetricHistory{
						Name:           "metric1",
						Type:           "uint",
						NumAggregation: 1,
						DisplayOrder:   0,
						IsAlarmEnabled: true,
						History: []common.MetricValue{
							{
								Value:      "1",
								RecordedAt: time.Now().Unix(),
							},
						},
					}

					metric2 := common.MetricHistory{
						Name:           "metric2",
						Type:           "uint",
						NumAggregation: 1,
						DisplayOrder:   0,
						IsAlarmEnabled: true,
						History:        nil,
					}

					return []common.MetricHistory{metric1, metric2}, nil
				},
			},
			&testsCommon.OutputNotifiersHandlerStub{
				NotifyWithRetryHandler: func(caller string, messages ...common.OutputMessage) error {
					assert.Equal(t, 1, len(messages))
					assert.Equal(t, "metric2", messages[0].Identifier)
					atomic.AddUint32(&notifyWithRetryNumCalled, 1)

					return nil
				},
			},
			&testsCommon.StatusHandlerStub{
				CollectKeysProblemsHandler: func(messages []common.OutputMessage) {
					assert.Equal(t, 1, len(messages))
					assert.Equal(t, "metric2", messages[0].Identifier)

					atomic.AddUint32(&collectKeysProblemsNumCalled, 1)
				},
			},
			1,
			time.Millisecond*100)

		alarm.Start()
		defer func() {
			_ = alarm.Close()
		}()

		time.Sleep(time.Millisecond * 350)

		assert.Equal(t, uint32(1), atomic.LoadUint32(&notifyWithRetryNumCalled))
		assert.Equal(t, uint32(1), atomic.LoadUint32(&collectKeysProblemsNumCalled))
	})
	t.Run("one metric without history but not alarm enabled should not notify", func(t *testing.T) {
		t.Parallel()

		notifyWithRetryNumCalled := uint32(0)
		collectKeysProblemsNumCalled := uint32(0)

		alarm, _ := NewAlarmService(
			&testsCommon.StoreStub{
				GetLatestMetricsHandler: func(ctx context.Context) ([]common.MetricHistory, error) {
					metric1 := common.MetricHistory{
						Name:           "metric1",
						Type:           "uint",
						NumAggregation: 1,
						DisplayOrder:   0,
						IsAlarmEnabled: true,
						History: []common.MetricValue{
							{
								Value:      "1",
								RecordedAt: time.Now().Unix(),
							},
						},
					}

					metric2 := common.MetricHistory{
						Name:           "metric2",
						Type:           "uint",
						NumAggregation: 1,
						DisplayOrder:   0,
						IsAlarmEnabled: false,
						History:        nil,
					}

					return []common.MetricHistory{metric1, metric2}, nil
				},
			},
			&testsCommon.OutputNotifiersHandlerStub{
				NotifyWithRetryHandler: func(caller string, messages ...common.OutputMessage) error {
					atomic.AddUint32(&notifyWithRetryNumCalled, 1)

					return nil
				},
			},
			&testsCommon.StatusHandlerStub{
				CollectKeysProblemsHandler: func(messages []common.OutputMessage) {
					atomic.AddUint32(&collectKeysProblemsNumCalled, 1)
				},
			},
			1,
			time.Millisecond*100)

		alarm.Start()
		defer func() {
			_ = alarm.Close()
		}()

		time.Sleep(time.Millisecond * 350)

		assert.Equal(t, uint32(0), atomic.LoadUint32(&notifyWithRetryNumCalled))
		assert.Equal(t, uint32(0), atomic.LoadUint32(&collectKeysProblemsNumCalled))
	})
	t.Run("one metric with stalled history should notify once", func(t *testing.T) {
		t.Parallel()

		notifyWithRetryNumCalled := uint32(0)
		collectKeysProblemsNumCalled := uint32(0)

		alarm, _ := NewAlarmService(
			&testsCommon.StoreStub{
				GetLatestMetricsHandler: func(ctx context.Context) ([]common.MetricHistory, error) {
					metric1 := common.MetricHistory{
						Name:           "metric1",
						Type:           "uint",
						NumAggregation: 1,
						DisplayOrder:   0,
						IsAlarmEnabled: true,
						History: []common.MetricValue{
							{
								Value:      "1",
								RecordedAt: time.Now().Add(-time.Second * 101).Unix(),
							},
						},
					}

					return []common.MetricHistory{metric1}, nil
				},
			},
			&testsCommon.OutputNotifiersHandlerStub{
				NotifyWithRetryHandler: func(caller string, messages ...common.OutputMessage) error {
					assert.Equal(t, 1, len(messages))
					assert.Equal(t, "metric1", messages[0].Identifier)
					atomic.AddUint32(&notifyWithRetryNumCalled, 1)

					return nil
				},
			},
			&testsCommon.StatusHandlerStub{
				CollectKeysProblemsHandler: func(messages []common.OutputMessage) {
					assert.Equal(t, 1, len(messages))
					assert.Equal(t, "metric1", messages[0].Identifier)

					atomic.AddUint32(&collectKeysProblemsNumCalled, 1)
				},
			},
			100,
			time.Millisecond*100)

		alarm.Start()
		defer func() {
			_ = alarm.Close()
		}()

		time.Sleep(time.Millisecond * 350)

		assert.Equal(t, uint32(1), atomic.LoadUint32(&notifyWithRetryNumCalled))
		assert.Equal(t, uint32(1), atomic.LoadUint32(&collectKeysProblemsNumCalled))
	})
}
