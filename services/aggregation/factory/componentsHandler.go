package factory

import (
	"github.com/iulianpascalau/mx-api-monitoring/services/aggregation/api"
	"github.com/iulianpascalau/mx-api-monitoring/services/aggregation/config"
	"github.com/iulianpascalau/mx-api-monitoring/services/aggregation/storage"
)

type componentsHandler struct {
	store  api.Storage
	server Server
}

// NewComponentsHandler creates a new components handler
func NewComponentsHandler(
	sqlitePath string,
	serviceKeyApi string,
	authUsername string,
	authPassword string,
	cfg config.Config,
) (*componentsHandler, error) {
	store, err := storage.NewSQLiteStorage(sqlitePath, cfg.RetentionSeconds)
	if err != nil {
		return nil, err
	}

	serverArgs := api.ArgsWebServer{
		ServiceKeyApi:  serviceKeyApi,
		AuthUsername:   authUsername,
		AuthPassword:   authPassword,
		ListenAddress:  cfg.ListenAddress,
		Storage:        store,
		GeneralHandler: api.CORSMiddleware,
	}

	server, err := api.NewServer(serverArgs)
	if err != nil {
		return nil, err
	}

	return &componentsHandler{
		store:  store,
		server: server,
	}, nil
}

// GetStore returns the storage component
func (ch *componentsHandler) GetStore() api.Storage {
	return ch.store
}

// GetServer returns the server component
func (ch *componentsHandler) GetServer() Server {
	return ch.server
}

// Start starts the inner components
func (ch *componentsHandler) Start() {
	ch.server.Start()
}

// Close closes the inner components
func (ch *componentsHandler) Close() {
	_ = ch.server.Close()
	_ = ch.store.Close()
}
