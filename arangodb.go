package arangodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	driver "github.com/arangodb/go-driver"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/template"
)

const (
	arangoDBTypeName        = "arangodb"
	defaultUserNameTemplate = `{{ printf "v-%s-%s-%s-%s" (.DisplayName | truncate 15) (.RoleName | truncate 15) (random 20) (unix_time) | truncate 100 }}`
)

// ArangoDB is an implementation of Database interface
type ArangoDB struct {
	*arangoDBConnectionProducer

	usernameProducer template.StringTemplate
}

type databaseGrant struct {
	Db     string       `json:"db"`
	Access driver.Grant `json:"access"`
}

type collectionGrant struct {
	Db         string       `json:"db"`
	Collection string       `json:"collection,omitempty"`
	Access     driver.Grant `json:"access"`
}

type userCreateStatement struct {
	DatabaseGrants   []databaseGrant   `json:"database_grants"`
	CollectionGrants []collectionGrant `json:"collection_grants"`
}

var _ dbplugin.Database = &ArangoDB{}

// New returns a new ArangoDB instance
func New() (interface{}, error) {
	db := Connect()
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.secretValues)

	return dbType, nil
}

func Connect() *ArangoDB {
	connProducer := &arangoDBConnectionProducer{}
	connProducer.Type = arangoDBTypeName

	return &ArangoDB{
		arangoDBConnectionProducer: connProducer,
	}
}

// Initialize sets up the ArangoDB Plugin
func (a *ArangoDB) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	usernameTemplate, err := strutil.GetString(req.Config, "username_template")
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to retrieve username_template: %w", err)
	}
	if usernameTemplate == "" {
		usernameTemplate = defaultUserNameTemplate
	}

	up, err := template.NewTemplate(template.Template(usernameTemplate))
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("unable to initialize username template: %w", err)
	}
	a.usernameProducer = up

	return a.arangoDBConnectionProducer.Initialize(ctx, req)
}

// Type returns the TypeName for this backend
func (a *ArangoDB) Type() (string, error) {
	return arangoDBTypeName, nil
}

// DeleteUser deletes a user account
func (a *ArangoDB) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	user, err := a.client.User(ctx, req.Username)
	if err != nil {
		return dbplugin.DeleteUserResponse{}, err
	}

	err = user.Remove(ctx)
	return dbplugin.DeleteUserResponse{}, err
}

// NewUser creates a new user account
func (a *ArangoDB) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	if len(req.Statements.Commands) == 0 {
		return dbplugin.NewUserResponse{}, dbutil.ErrEmptyCreationStatement
	}

	username, err := a.usernameProducer.Generate(req.UsernameConfig)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	// Unmarshal statements.CreationStatements
	var cs userCreateStatement
	err = json.Unmarshal([]byte(req.Statements.Commands[0]), &cs)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	// Create the user record itself
	options := driver.UserOptions{
		Password: req.Password,
	}

	user, err := a.client.CreateUser(ctx, username, &options)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	// Assign database grants
	for _, grant := range cs.DatabaseGrants {
		database, err := a.client.Database(ctx, grant.Db)
		if err != nil {
			return dbplugin.NewUserResponse{}, err
		}

		err = user.SetDatabaseAccess(ctx, database, grant.Access)
		if err != nil {
			return dbplugin.NewUserResponse{}, err
		}
	}

	// Assign collection grants
	for _, grant := range cs.CollectionGrants {
		database, err := a.client.Database(ctx, grant.Db)
		if err != nil {
			return dbplugin.NewUserResponse{}, err
		}

		// Set the default grant if no specific collection was specified
		if grant.Collection == "" {
			err = user.SetCollectionAccess(ctx, database, grant.Access)
			if err != nil {
				return dbplugin.NewUserResponse{}, err
			}
		} else {
			collection, err := database.Collection(ctx, grant.Collection)
			if err != nil {
				return dbplugin.NewUserResponse{}, err
			}

			err = user.SetCollectionAccess(ctx, collection, grant.Access)
			if err != nil {
				return dbplugin.NewUserResponse{}, err
			}
		}
	}

	resp := dbplugin.NewUserResponse{
		Username: user.Name(),
	}

	return resp, nil
}

// UpdateUser updates a user account
func (a *ArangoDB) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	return dbplugin.UpdateUserResponse{}, errors.New("Not implemented")
}
