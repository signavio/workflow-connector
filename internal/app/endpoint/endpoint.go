package endpoint

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
)

var (
	ErrPostForm               = errors.New("Form data sent was empty and/or not of type `application/x-www-form-urlencoded`")
	ErrCardinalityMany        = errors.New("Form data contained multiple input values for a single column")
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
)

// Endpoint fetches data from a backend (ie. a SQL DB) and makes the data
// available via a standard REST API to the user.

type Endpoint interface {
	CRUD
	DataConnectorOptions
	// GetHandler returns a http.Handler which include all the routes that implement
	// the functionality required by the CRUD and WorkflowConnector interfaces
	GetHandler() http.Handler
	Open(...interface{}) error
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

// DataConnectorOptions satisfies the interface for a custom connector as
// specified by Signavio's Workflow Accelerator documentation
type DataConnectorOptions interface {
	GetSingleAsOption(rw http.ResponseWriter, req *http.Request)
	GetCollectionAsOptions(rw http.ResponseWriter, req *http.Request)
}

type Backend interface {
	GetSchemaMapping(string) *descriptor.SchemaMapping
	SaveSchemaMapping() error
	GetQueryTemplate(string) string
	QueryContext(context.Context, string, ...interface{}) ([]interface{}, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	CommitTx(string) error
	CreateTx(time.Duration) (uuid.UUID, error)
}
