package factory

import (
	"context"
	"sync"
	"time"

	"github.com/iulianpascalau/api-monitoring/commonGo"
	"github.com/iulianpascalau/api-monitoring/services/agent/config"
	"github.com/iulianpascalau/api-monitoring/services/agent/engine"
	"github.com/iulianpascalau/api-monitoring/services/agent/poller"
	"github.com/iulianpascalau/api-monitoring/services/agent/reporter"
)

type componentsHandler struct {
	poller        engine.Poller
	reporter      engine.Reporter
	engine        Engine
	mutCancel     sync.Mutex
	cancel        func()
	queryInterval time.Duration
}

// NewComponentsHandler creates a new components handler
func NewComponentsHandler(
	serviceKeyApi string,
	cfg config.Config,
) (*componentsHandler, error) {
	poll := poller.NewHTTPPoller(time.Duration(cfg.QueryIntervalInSeconds) * time.Second)
	rep := reporter.NewHTTPReporter(cfg.ReportEndpoint, serviceKeyApi, cfg.Name, time.Duration(cfg.ReportTimeoutInSeconds)*time.Second)

	eng, err := engine.NewAgentEngine(cfg, poll, rep)
	if err != nil {
		return nil, err
	}

	return &componentsHandler{
		poller:        poll,
		reporter:      rep,
		engine:        eng,
		queryInterval: time.Duration(cfg.QueryIntervalInSeconds) * time.Second,
	}, nil
}

// GetPoller returns the poller component
func (ch *componentsHandler) GetPoller() engine.Poller {
	return ch.poller
}

// GetReporter returns the reporter component
func (ch *componentsHandler) GetReporter() engine.Reporter {
	return ch.reporter
}

// GetEngine returns the engine component
func (ch *componentsHandler) GetEngine() Engine {
	return ch.engine
}

// Start starts the inner components
func (ch *componentsHandler) Start() {
	ch.mutCancel.Lock()
	defer ch.mutCancel.Unlock()

	if ch.cancel != nil {
		return
	}

	var ctx context.Context
	ctx, ch.cancel = context.WithCancel(context.Background())

	commonGo.CronJobStarter(ctx, ch.engine.Process, ch.queryInterval)
}

// Close closes the inner components
func (ch *componentsHandler) Close() {
	ch.mutCancel.Lock()
	defer ch.mutCancel.Unlock()

	if ch.cancel == nil {
		return
	}

	ch.cancel()
	ch.cancel = nil
}
