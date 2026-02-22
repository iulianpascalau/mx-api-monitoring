package factory

import (
	"fmt"
	"testing"

	"github.com/iulianpascalau/mx-api-monitoring/services/agent/config"
	"github.com/stretchr/testify/assert"
)

func TestNewComponentsHandler(t *testing.T) {
	t.Parallel()

	handler, err := NewComponentsHandler(
		"service-key",
		config.Config{
			Name:                   "vm1",
			QueryIntervalInSeconds: 1,
			ReportEndpoint:         "/report",
			ReportTimeoutInSeconds: 1,
			Endpoints:              nil,
		})

	assert.NotNil(t, handler)
	assert.Nil(t, err)

	handler.Close()
}

func TestComponentsHandlerMethods(t *testing.T) {
	t.Parallel()

	handler, _ := NewComponentsHandler(
		"service-key",
		config.Config{
			Name:                   "vm1",
			QueryIntervalInSeconds: 1,
			ReportEndpoint:         "/report",
			ReportTimeoutInSeconds: 1,
			Endpoints:              nil,
		})

	handler.Start()

	poller := handler.GetPoller()
	assert.Equal(t, "*poller.httpPoller", fmt.Sprintf("%T", poller))

	reporter := handler.GetReporter()
	assert.Equal(t, "*reporter.httpReporter", fmt.Sprintf("%T", reporter))

	engine := handler.GetEngine()
	assert.Equal(t, "*engine.agentEngine", fmt.Sprintf("%T", engine))

	handler.Close()
}
