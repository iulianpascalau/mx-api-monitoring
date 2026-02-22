package factory

import "context"

// Engine defines the agent's operations
type Engine interface {
	Process(ctx context.Context)
	IsInterfaceNil() bool
}
