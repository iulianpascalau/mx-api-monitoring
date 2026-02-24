package engine

import (
	"context"
	"errors"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/agent/config"
	"github.com/multiversx/mx-chain-core-go/core/check"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("engine")

// agentEngine orchestrates polling and reporting at configured intervals
type agentEngine struct {
	config   config.Config
	poller   Poller
	reporter Reporter
}

// NewAgentEngine creates a new engine instance
func NewAgentEngine(cfg config.Config, p Poller, r Reporter) (*agentEngine, error) {
	if check.IfNil(p) {
		return nil, errors.New("nil poller")
	}
	if check.IfNil(r) {
		return nil, errors.New("nil reporter")
	}

	return &agentEngine{
		config:   cfg,
		poller:   p,
		reporter: r,
	}, nil
}

// Process will poll all endpoints and try to send the report to the reporter
func (e *agentEngine) Process(ctx context.Context) {
	log.Debug("waking up to poll endpoints", "count", len(e.config.Endpoints))

	// 1. Poll all endpoints concurrently
	pollCtx, cancelPoll := context.WithTimeout(ctx, 30*time.Second) // Prevent indefinite hanging
	defer cancelPoll()
	results := e.poller.PollAll(pollCtx, e.config.Endpoints)

	log.Debug("finished polling", "successful_results", len(results))

	// 2. Report them to aggregation backend
	reportCtx, cancelReport := context.WithTimeout(ctx, 10*time.Second)
	defer cancelReport()

	err := e.reporter.Report(reportCtx, results)
	if err != nil {
		log.Warn("failed to report metrics, they will be discarded", "error", err)
	}
}

// IsInterfaceNil returns true if the value under the interface is nil
func (e *agentEngine) IsInterfaceNil() bool {
	return e == nil
}
