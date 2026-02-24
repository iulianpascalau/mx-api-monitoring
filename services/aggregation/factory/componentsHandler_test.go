package factory

import (
	"fmt"
	"testing"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/config"
	"github.com/stretchr/testify/assert"
)

func TestNewComponentsHandler(t *testing.T) {
	t.Parallel()

	handler, err := NewComponentsHandler(
		":memory:",
		"service-key",
		"admin",
		"admin123",
		config.Config{
			ListenAddress:    "0.0.0.0:0",
			RetentionSeconds: 3600,
		})

	assert.NotNil(t, handler)
	assert.Nil(t, err)

	handler.Close()
}

func TestComponentsHandlerMethods(t *testing.T) {
	t.Parallel()

	handler, _ := NewComponentsHandler(
		":memory:",
		"service-key",
		"admin",
		"admin123",
		config.Config{
			ListenAddress:    "0.0.0.0:0",
			RetentionSeconds: 3600,
		})

	handler.Start()

	store := handler.GetStore()
	assert.Equal(t, "*storage.sqliteStorage", fmt.Sprintf("%T", store))

	serv := handler.GetServer()
	assert.Equal(t, "*api.server", fmt.Sprintf("%T", serv))

	handler.Close()
}
