package engine

import (
	"testing"

	"github.com/iulianpascalau/api-monitoring/services/agent/config"
	"github.com/iulianpascalau/api-monitoring/services/agent/testsCommon"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentEngine(t *testing.T) {
	t.Parallel()

	t.Run("nil poller should error", func(t *testing.T) {
		engine, err := NewAgentEngine(config.Config{}, nil, &testsCommon.ReporterStub{})

		assert.Nil(t, engine)
		assert.True(t, engine.IsInterfaceNil())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil poller")
	})
	t.Run("nil reporter should error", func(t *testing.T) {
		engine, err := NewAgentEngine(config.Config{}, &testsCommon.PollerStub{}, nil)

		assert.Nil(t, engine)
		assert.True(t, engine.IsInterfaceNil())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil reporter")
	})
	t.Run("should work", func(t *testing.T) {
		engine, err := NewAgentEngine(config.Config{}, &testsCommon.PollerStub{}, &testsCommon.ReporterStub{})

		assert.NotNil(t, engine)
		assert.False(t, engine.IsInterfaceNil())
		assert.Nil(t, err)
	})
}
