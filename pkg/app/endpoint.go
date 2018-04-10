package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/pkg/config"
	sqlBackend "github.com/signavio/workflow-connector/pkg/sql"
	"github.com/signavio/workflow-connector/pkg/sql/mssql"
	"github.com/signavio/workflow-connector/pkg/sql/mysql"
	"github.com/signavio/workflow-connector/pkg/sql/pgsql"
	"github.com/signavio/workflow-connector/pkg/sql/sqlite"
)

// Endpoint fetches data from a backend service and makes the data available via
// a standard REST API to the user. It also defines specific routes which
// provide additional functionality, such as being able to wrap a
// group of CRUD calls within a DB Transaction

type Endpoint interface {
	BackendService
}

// BackendService communicates with a particular backend, for example, with a SQL
// Server, to provide basic CRUD functionality.
type BackendService interface {
	CRUD
	WorkflowConnector
}

// CRUD abstracts the functionality expected from a standard CRUD service,
// this includes methods for creating, reading, updating and deleting resources.
type CRUD interface {
	CreateSingle(req *http.Request) (response []interface{}, err error)
	GetSingle(req *http.Request) (response []interface{}, err error)
	GetCollection(req *http.Request) (response []interface{}, err error)
	UpdateSingle(req *http.Request) (response []interface{}, err error)
	// TODO
	//Delete(req *http.Request) (response []interface{}, err error)
}

// WorkflowConnector satisfies the interface for a custom connector as
// specified by Signavio's Workflow Accelerator documentation
type WorkflowConnector interface {
	GetSingleAsOption(req *http.Request) (response []interface{}, err error)
	GetCollectionAsOptions(req *http.Request) (response []interface{}, err error)
	GetCollectionAsOptionsFilterable(req *http.Request) (response []interface{}, err error)
}

// Formatter takes the results it receives from the Backend Service and
// converts those into a JSON Format that Signavio's Workflow Accelerator
// can properly parse.
type Formatter interface {
	Format(ctx context.Context, cfg config.Config, results []interface{}) (JSONResults []byte, err error)
}

var (
	ErrPostForm               = errors.New("Form data sent was empty and/or not of type `application/x-www-form-urlencoded`")
	ErrCardinalityMany        = errors.New("Form data contained multiple input values for a single column")
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
)

func NewEndpoint(cfg *config.Config, router *mux.Router) (Endpoint, error) {
	switch cfg.Database.Driver {
	case "sqlserver":
		return connectToBackend(mssql.NewMssqlBackend, router, cfg)
	case "sqlite3":
		return connectToBackend(sqlite.NewSqliteBackend, router, cfg)
	case "mysql":
		return connectToBackend(mysql.NewMysqlBackend, router, cfg)
	case "postgres":
		return connectToBackend(pgsql.NewPgsqlBackend, router, cfg)
	default:
		return nil, fmt.Errorf("Database driver: %s, not supported", cfg.Database.Driver)

	}
}
func connectToBackend(backendFn func(*config.Config, *mux.Router) *sqlBackend.Backend, router *mux.Router, cfg *config.Config) (*sqlBackend.Backend, error) {
	backend := backendFn(cfg, router)
	err := backend.Open(cfg.Database.Driver, cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	err = backend.SaveTableSchemas()
	if err != nil {
		return nil, err
	}
	return backend, nil
}
