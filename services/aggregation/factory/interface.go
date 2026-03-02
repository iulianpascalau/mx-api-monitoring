package factory

// Server defines the operation of an entity able to serve requests
type Server interface {
	Start()
	Close() error
	Address() string
}

// AlarmEngine defines the operations of an entity able to send alarms
type AlarmEngine interface {
	Start()
	Close() error
	IsInterfaceNil() bool
}

// PollingHandler defines the operations of an entity able to poll for data
type PollingHandler interface {
	StartProcessingLoop() error
	IsRunning() bool
	Close() error
	IsInterfaceNil() bool
}
