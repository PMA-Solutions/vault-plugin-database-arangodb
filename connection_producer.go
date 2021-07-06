package arangodb

import (
	"context"
	"fmt"
	"sync"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/mitchellh/mapstructure"
)

// arangoDBConnectionProducer implements ConnectionProducer and provides an
// interface for databases to make connections.
type arangoDBConnectionProducer struct {
	Username      string `json:"username" structs:"username" mapstructure:"username"`
	Password      string `json:"password" structs:"password" mapstructure:"password"`
	ConnectionURL string `json:"connection_url" structs:"connection_url" mapstructure:"connection_url"`

	rawConfig map[string]interface{}

	Initialized bool
	Type        string
	client      driver.Client
	sync.Mutex
}

func (a *arangoDBConnectionProducer) secretValues() map[string]string {
	return map[string]string{
		a.Password: "[password]",
		a.Password: "[password]",
	}
}

func (a *arangoDBConnectionProducer) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	a.Lock()
	defer a.Unlock()

	a.rawConfig = req.Config

	err := mapstructure.WeakDecode(req.Config, a)
	if err != nil {
		return dbplugin.InitializeResponse{}, err
	}

	// Set initialized to true at this point since all fields are set,
	// and the connection can be established at a later time.
	a.Initialized = true

	if req.VerifyConnection {
		_, err := a.Connection(ctx)
		if err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("failed to verify connection: %w", err)
		}

		_, err = a.client.Version(ctx)
		if err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("failed to verify connection: %w", err)
		}
	}

	resp := dbplugin.InitializeResponse{
		Config: req.Config,
	}

	return resp, nil
}

// Connection creates a database connection
func (a *arangoDBConnectionProducer) Connection(ctx context.Context) (interface{}, error) {
	if !a.Initialized {
		return nil, connutil.ErrNotInitialized
	}

	// If we already have a DB, return it
	if a.client != nil {
		return a.client, nil
	}

	client, err := a.createClient()
	if err != nil {
		return nil, err
	}

	//  Store the session in backend for reuse
	a.client = client

	return a.client, nil
}

func (a *arangoDBConnectionProducer) createClient() (driver.Client, error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{a.ConnectionURL},
	})
	if err != nil {
		return nil, err
	}

	auth := driver.BasicAuthentication(a.Username, a.Password)

	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: auth,
	})

	if err != nil {
		return nil, err
	}

	return c, nil
}

// Close terminates the database connection.
func (a *arangoDBConnectionProducer) Close() error {
	a.Lock()
	defer a.Unlock()

	a.client = nil

	return nil
}
