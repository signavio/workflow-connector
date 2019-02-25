// Package backend defines a Backend that is responsible for communicating
// with SQL databases and other external systems
package backend

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/filter"
)

var (
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
	ErrNoLastInsertID         = errors.New("Database does not support getting the last inserted ID")
)

type Backend struct {
	GetSchemaMappingFunc                 func(string) *descriptor.SchemaMapping
	GetFilterPredicateMappingFunc        func(filter.Predicate) string
	GetQueryTemplateFunc                 func(string) string
	CoerceExecArgsFunc                   func(string, []string, []*descriptor.Field) string
	CastBackendTypeToGolangType         func(string) interface{}
	QueryContextFunc                     func(context.Context, string, ...interface{}) ([]interface{}, error)
	ExecContextFunc                      func(context.Context, string, ...interface{}) (sql.Result, error)
	ExtractAndFormatQueryParamsAndValues func(string, url.Values) (map[string]string, error)
	OpenFunc                             func(...interface{}) error
	CreateTxFunc                         func(time.Duration) (uuid.UUID, error)
	CommitTxFunc                         func(string) error
}

func appendHandlers(r *mux.Router, b *Backend) *mux.Router {
	r.HandleFunc("/{table}/options/{id}", b.GetSingleAsOption).
		Name("GetSingleAsOption").
		Methods("GET")
	r.HandleFunc("/{table}/options", b.GetCollectionAsOptionsWithParams).
		Name("GetCollectionAsOptionsWithParams").
		Methods("GET").
		Queries("filter", "{filter}").
		MatcherFunc(func(req *http.Request, rm *mux.RouteMatch) bool {
			return len(req.URL.Query()) > 1
		})
	r.HandleFunc("/{table}/options", b.GetCollectionAsOptionsFilterable).
		Name("GetCollectionAsOptionsFilterable").
		Methods("GET").
		Queries("filter", "{filter}")
	r.HandleFunc("/{table}/options", b.GetCollectionAsOptions).
		Name("GetCollectionAsOptions").
		Methods("GET")
	r.HandleFunc("/{table}/{id}", b.GetSingle).
		Name("GetSingle").
		Methods("GET")
	r.HandleFunc("/{table}/{id}", b.UpdateSingle).
		Name("UpdateSingle").
		Methods("PATCH")
	r.HandleFunc("/{table}/{id}", b.UpdateSingle).
		Name("UpdateSingle").
		Methods("PATCH").
		Queries("tx", "{tx}")
	r.HandleFunc("/{table}", b.GetCollectionFilterable).
		Name("GetCollectionFilterable").
		Methods("GET").
		Queries("filter", "{filter}")
	r.HandleFunc("/{table}", b.GetCollection).
		Name("GetCollection").
		Methods("GET")
	r.HandleFunc("/{table}", b.CreateSingle).
		Name("CreateSingle").
		Methods("POST")
	r.HandleFunc("/{table}", b.CreateSingle).
		Name("CreateSingle").
		Methods("POST").Queries("tx", "{tx}")
	r.HandleFunc("/{table}/{id}", b.DeleteSingle).
		Name("DeleteSingle").
		Methods("DELETE").
		Queries("tx", "{tx}")
	r.HandleFunc("/{table}/{id}", b.DeleteSingle).
		Name("DeleteSingle").
		Methods("DELETE")
	r.HandleFunc("/", b.GetDescriptorFile).
		Name("GetDescriptorFile").
		Methods("GET")
	r.HandleFunc("/", b.CreateTransaction).
		Name("CreateTx").
		Methods("POST").
		Queries("begin", "{begin}")
	r.HandleFunc("/", b.CommitTransaction).
		Name("CommitTx").
		Methods("POST").
		Queries("commit", "{commit}")
	return r
}
func (b *Backend) GetHandler() http.Handler {
	r := mux.NewRouter()
	return appendHandlers(r, b)
}

// Open a connection to the backend database
func (b *Backend) Open(args ...interface{}) error {
	return b.OpenFunc(args...)
}

func (b *Backend) GetCoerceExecArgsFunc() func(string, []string, []*descriptor.Field) string {
	return b.CoerceExecArgsFunc
}

func (b *Backend) GetQueryTemplate(name string) string {
	return b.GetQueryTemplateFunc(name)
}

func (b *Backend) GetFilterPredicateMapping(predicate filter.Predicate) string {
	return b.GetFilterPredicateMappingFunc(predicate)
}
func (b *Backend) GetSchemaMapping(typeDescriptor string) *descriptor.SchemaMapping {
	return b.GetSchemaMappingFunc(typeDescriptor)
}

func (b *Backend) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	return b.ExecContextFunc(ctx, query, args...)
}

func (b *Backend) QueryContext(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	return b.QueryContextFunc(ctx, query, args...)
}

func (b *Backend) CommitTx(txUUID string) (err error) {
	return b.CommitTxFunc(txUUID)
}

func (b *Backend) CreateTx(timeout time.Duration) (txUUID uuid.UUID, err error) {
	return b.CreateTxFunc(timeout)
}
