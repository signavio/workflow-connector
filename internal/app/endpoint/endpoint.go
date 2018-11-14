package endpoint

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/backend/sql"
	"github.com/signavio/workflow-connector/internal/pkg/config"
)

// Endpoint fetches data from a backend (ie. a SQL DB) and makes the data
// available via a standard REST API to the user.

type Endpoint interface {
	CRUD
	WorkflowConnector
	// GetHandler returns a http.Handler which include all the routes that implement
	// the functionality required by the CRUD and WorkflowConnector interfaces
	GetHandler() http.Handler
	Open(args ...interface{}) error
}

// CRUD abstracts the functionality expected from a standard CRUD service,
// this includes methods for creating, reading, updating and deleting resources.
type CRUD interface {
	CreateSingle(rw http.ResponseWriter, req *http.Request)
	GetSingle(rw http.ResponseWriter, req *http.Request)
	GetCollection(rw http.ResponseWriter, req *http.Request)
	UpdateSingle(rw http.ResponseWriter, req *http.Request)
	DeleteSingle(rw http.ResponseWriter, req *http.Request)
}

// WorkflowConnector satisfies the interface for a custom connector as
// specified by Signavio's Workflow Accelerator documentation
type WorkflowConnector interface {
	GetSingleAsOption(rw http.ResponseWriter, req *http.Request)
	GetCollectionAsOptions(rw http.ResponseWriter, req *http.Request)
	GetCollectionAsOptionsFilterable(rw http.ResponseWriter, req *http.Request)
}

var (
	ErrPostForm               = errors.New("Form data sent was empty and/or not of type `application/x-www-form-urlencoded`")
	ErrCardinalityMany        = errors.New("Form data contained multiple input values for a single column")
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
)

func NewEndpoint(cfg config.Config) (Endpoint, error) {
	switch cfg.Database.Driver {
	case "sqlserver":
		return sql.NewBackend("sqlserver"), nil
	case "sqlite":
		return sql.NewBackend("sqlite"), nil
	case "mysql":
		return sql.NewBackend("mysql"), nil
	case "postgres":
		return sql.NewBackend("postgres"), nil
	case "goracle":
		return sql.NewBackend("goracle"), nil
	default:
		return nil, fmt.Errorf("Database driver: %s, not supported", cfg.Database.Driver)

	}
}
