package alarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/multiversx/mx-chain-core-go/core/check"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("alarm")

const (
	alarmMessage = "Host appears offline"
)

type alarmService struct {
	numSecondsToConsiderStale uint32
	store                     Storage
	outputNotifiersHandler    OutputNotifiersHandler
	statusHandler             StatusHandler

	mutCancel  sync.Mutex
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	loopTime   time.Duration

	// Mutex and map to avoid spamming the same alarm
	// Maps a metric name to the last time we triggered an alarm
	mutTriggered     sync.Mutex
	triggeredMetrics map[string]bool
}

// NewAlarmService creates a new alarm service
func NewAlarmService(
	store Storage,
	outputNotifiersHandler OutputNotifiersHandler,
	statusHandler StatusHandler,
	numSecondsToConsiderStale uint32,
	loopTime time.Duration,
) (*alarmService, error) {
	if check.IfNil(store) {
		return nil, fmt.Errorf("nil storage provided to alarm service")
	}
	if check.IfNil(outputNotifiersHandler) {
		return nil, fmt.Errorf("nil output notifiers handler provided to alarm service")
	}
	if check.IfNil(statusHandler) {
		return nil, fmt.Errorf("nil status handler provided to alarm service")
	}
	if numSecondsToConsiderStale == 0 {
		return nil, fmt.Errorf("num seconds to consider stale must be greater than 0")
	}
	if loopTime < time.Millisecond*10 {
		return nil, fmt.Errorf("loop time must be greater than 10ms")
	}

	return &alarmService{
		numSecondsToConsiderStale: numSecondsToConsiderStale,
		store:                     store,
		outputNotifiersHandler:    outputNotifiersHandler,
		statusHandler:             statusHandler,
		loopTime:                  loopTime,
		triggeredMetrics:          make(map[string]bool),
	}, nil
}

// Start spawns the background goroutine that periodically checks metrics
func (as *alarmService) Start() {
	as.mutCancel.Lock()
	defer as.mutCancel.Unlock()

	// won't start twice
	if as.cancelFunc != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	as.cancelFunc = cancel

	as.wg.Add(1)
	go as.runLoop(ctx)
}

// Close stops the background routine
func (as *alarmService) Close() error {
	as.mutCancel.Lock()
	defer as.mutCancel.Unlock()

	if as.cancelFunc == nil {
		return nil
	}

	as.cancelFunc()
	as.wg.Wait()
	as.cancelFunc = nil

	return nil
}

func (as *alarmService) runLoop(ctx context.Context) {
	defer as.wg.Done()

	ticker := time.NewTicker(as.loopTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Alarm service shutting down...")
			return
		case <-ticker.C:
			as.checkMetrics(ctx)
		}
	}
}

func (as *alarmService) checkMetrics(ctx context.Context) {
	metrics, err := as.store.GetLatestMetrics(ctx)
	if err != nil {
		log.Error("alarm service failed to fetch latest metrics", "error", err)
		return
	}

	log.Debug("alarm service fetched latest metrics to be checked", "num metrics", len(metrics))

	metricsToNotify := make([]common.MetricHistory, 0)
	for _, m := range metrics {
		if !as.shouldNotify(m) {
			continue
		}

		metricsToNotify = append(metricsToNotify, m)
	}

	if len(metricsToNotify) > 0 {
		as.triggerAlarm(metricsToNotify, alarmMessage)
	}
}

func (as *alarmService) shouldNotify(metric common.MetricHistory) bool {
	if !metric.IsAlarmEnabled {
		return false
	}

	stale := as.isMetricStale(metric)
	as.mutTriggered.Lock()
	oldStale := as.triggeredMetrics[metric.Name]
	as.triggeredMetrics[metric.Name] = stale
	as.mutTriggered.Unlock()

	if oldStale == stale {
		// nothing was changed, we should not notify
		return false
	}

	return stale
}

func (as *alarmService) isMetricStale(metric common.MetricHistory) bool {
	if len(metric.History) == 0 {
		return true
	}

	lastVal := metric.History[0]
	nowSec := time.Now().Unix()
	diffSec := nowSec - lastVal.RecordedAt

	return uint32(diffSec) >= as.numSecondsToConsiderStale
}

func (as *alarmService) triggerAlarm(metricsToNotify []common.MetricHistory, problem string) {
	messages := make([]common.OutputMessage, 0, len(metricsToNotify))

	for _, metric := range metricsToNotify {
		msg := common.OutputMessage{
			Type:               common.ErrorMessageOutputType, // mapping to defined constants in common package
			Identifier:         metric.Name,
			ExecutorName:       common.ExecutorName,
			ProblemEncountered: problem,
		}

		messages = append(messages, msg)
	}

	_ = as.outputNotifiersHandler.NotifyWithRetry(fmt.Sprintf("%T", as), messages...)
	as.statusHandler.CollectKeysProblems(messages)
}

// IsInterfaceNil returns true if the value under the interface is nil
func (as *alarmService) IsInterfaceNil() bool {
	return as == nil
}
