package factory

// Server defines the operation of an entity able to serve requests
type Server interface {
	Start()
	Close() error
}
