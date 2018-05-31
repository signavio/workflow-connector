// Package sql defines a Backend that is responsible for communicating
// with SQL databases
package sql

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
	ErrNoLastInsertID         = errors.New("Database does not support getting the last inserted ID")
	addHandlers               = func(r *mux.Router, b *Backend) *mux.Router {
		r.HandleFunc("/{table}/options/{id}", b.GetSingleAsOption).
			Name("GetSingleAsOption").
			Methods("GET")
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
		r.HandleFunc("/{table}", b.GetCollection).
			Name("GetCollection").
			Methods("GET")
		r.HandleFunc("/{table}", b.CreateSingle).
			Name("CreateSingle").
			Methods("POST")
		r.HandleFunc("/{table}", b.CreateSingle).
			Name("CreateSingle").
			Methods("POST").
			Queries("tx", "{tx}")
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
		r.HandleFunc("/", b.CreateDBTransaction).
			Name("CreateDBTransaction").
			Methods("POST").
			Queries("begin", "{begin}")
		r.HandleFunc("/", b.CommitDBTransaction).
			Name("CommitDBTransaction").
			Methods("POST").
			Queries("commit", "{commit}")
		return r
	}
	commonTableSchema = &TableSchema{
		[]string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		[]interface{}{
			&sql.NullString{String: "", Valid: true},
			&sql.NullString{String: "", Valid: true},
			&sql.NullFloat64{Float64: 0, Valid: true},
			&sql.NullString{String: "", Valid: true},
		},
	}
	descriptorFileBase = `
{
  "key": "test",
  "name": "Test",
  "description": "Just a test",
  "typeDescriptors": [
    {
      "key" : "equipment",
      "name" : "Equipment",
      "tableName": "equipment",
      "columnAsOptionName": "name",
      "uniqueIdColumn": "id",
      "fields" : [
        %s
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ]
}
`
	commonDescriptorFields = `
{
  "key" : "id",
  "name" : "ID",
  "fromColumn": "id",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "name",
  "name" : "Equipment Name",
  "fromColumn": "name",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "acquisitionCost",
  "name" : "Acquisition Cost",
  "type" : {
	"name" : "money",
	"amount" : {
      "key": "acquisitionCost",
	  "fromColumn": "acquisition_cost"
	},
	"currency" : {
	  "value" : "EUR"
	}
  }
},
{
  "key" : "purchaseDate",
  "name" : "Purchase Date",
  "fromColumn" : "purchase_date",
  "type" : {
	"name" : "date",
	"kind" : "date"
  }
}`
	queryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"  FROM {{.TableName}} AS _{{.TableName}} " +
			"  {{range .Relations}}" +
			"     LEFT JOIN {{.Relationship.WithTable}}" +
			"     ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			"     = _{{$.TableName}}.{{.UniqueIDColumn}}" +
			"  {{end}}" +
			"  WHERE _{{$.TableName}}.{{$.UniqueIDColumn}} = ?",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE {{.UniqueIDColumn}} = ?",
		"GetCollection": "SELECT * " +
			"FROM {{.TableName}}",
		"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}}",
		"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE {{.ColumnAsOptionName}} LIKE ?",
		"UpdateSingle": "UPDATE {{.TableName}} SET {{.ColumnNames | head}}" +
			" = ?{{range .ColumnNames | tail}}, {{.}} = ?{{end}} WHERE {{.UniqueIDColumn}} = ?",
		"CreateSingle": "INSERT INTO {{.TableName}}({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}, {{.}}{{end}}) " +
			"VALUES(?{{range .ColumnNames | tail}}, ?{{end}})",
		"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = ?",
		"GetTableSchema": "SELECT * " +
			"FROM {{.TableName}} " +
			"LIMIT 1",
		"GetTableWithRelationshipsSchema": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.{{.UniqueIDColumn}}{{end}} LIMIT 1",
	}
)

// TableSchema stores the schema of the database in use
type TableSchema struct {
	ColumnNames []string
	DataTypes   []interface{}
}

type Backend struct {
	ConvertDBSpecificDataType func(string) interface{}
	DB                        *sql.DB
	TableSchemas              map[string]*TableSchema
	Templates                 map[string]string
	Transactions              sync.Map
	TransactDirectly          func(context.Context, *sql.DB, string, ...interface{}) (sql.Result, error)
	TransactWithinTx          func(context.Context, *sql.Tx, string, ...interface{}) (sql.Result, error)
}

type handler struct {
	vars         []string
	templateData interface{}
}

// TestCases for backend
type TestCase struct {
	// A TestCase should assert success cases or failure cases
	Kind string
	// A TestCase has a unique name
	Name string
	// A TestCase has descriptor fields that describe the schema of the
	// mocked database table in workflow accelerator's custom json format
	DescriptorFields string
	// A TestCase has a tableSchema that describes the schema of the
	// mocked database table using golang's native data types
	TableSchema *TableSchema
	// A TestCase contains an array with the names of all columns in the
	// mocked database table
	ColumnNames []string
	// A TestCase contains the row data for each column in the mocked database
	// table in csv format
	RowsAsCsv string
	// A TestCase contains the SQL queries which should be executed against
	// the mock database
	ExpectedQueries func(sqlmock.Sqlmock, []string, string, ...driver.Value)
	// A TestCase contains the expected results that should be returned after
	// the mocked database has been queried and the results are processed
	// by the formatter
	ExpectedResults string
	// A TestCase contains the test data that a client would submit in an
	// HTTP POST
	PostData url.Values
	// A TestCase contains a *http.Request
	Request *http.Request
	// run the testcase
	Run func(t *testing.T, tc TestCase, args ...interface{})
}

// NewBackend
func NewBackend() *Backend {
	return &Backend{
		TableSchemas: make(map[string]*TableSchema),
		Templates:    make(map[string]string),
	}
}

// Open a connection to the backend database
func (b *Backend) Open(args ...interface{}) error {
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		config.Options.Database.Driver,
	)
	driver := args[0].(string)
	url := args[1].(string)
	db, err := sql.Open(driver, url)
	if err != nil {
		return fmt.Errorf("Error opening connection to database: %s", err)
	}
	b.DB = db
	err = b.SaveTableSchemas()
	if err != nil {
		return fmt.Errorf("Error saving table schema: %s", err)
	}
	return nil
}

func (b *Backend) GetHandler() http.Handler {
	r := mux.NewRouter()
	return addHandlers(r, b)
}

func (b *Backend) SaveTableSchemas() (err error) {
	log.When(config.Options.Logging).Infoln(
		"[backend] query database and save table schemas",
	)
	for _, table := range config.Options.Database.Tables {
		if util.TableHasRelationships(config.Options, table.Name) {
			err := b.populateBackendTableSchemas(
				table.Name,
				"GetTableWithRelationshipsSchema",
				fmt.Sprintf("%s_relationships", table.Name),
				getTableSchemaWithRelationships,
			)
			if err != nil {
				return err
			}
		} else {
			err := b.populateBackendTableSchemas(
				table.Name,
				"GetTableSchema",
				table.Name,
				getTableSchema,
			)
			if err != nil {
				return err
			}
		}
	}
	log.When(config.Options.Logging).Infof(
		"[backend] the following table schemas were retrieved:\n%#+v\n",
		b.TableSchemas,
	)
	return nil
}

func (b *Backend) populateBackendTableSchemas(tableName, templateName, tableSchemaName string, getTableSchemaFn func(*Backend, string, string) (*TableSchema, error)) error {
	queryTemplate := b.Templates[templateName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName string
		}{
			TableName: tableName,
		},
	}
	query, err := handler.interpolateQueryTemplate()
	if err != nil {
		return err
	}
	b.TableSchemas[tableName], err = getTableSchemaFn(b, query, tableName)
	if err != nil {
		return fmt.Errorf("Unable to retrieve columns and data types from table schema: %s", err)
	}
	return nil
}

func getTableSchema(b *Backend, query, table string) (*TableSchema, error) {
	var dataTypes []interface{}
	var columnNames []string
	var columnsPrepended []string

	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	// Prepend all column names with table name. This makes it easier
	// to keep track of which column belongs to which table when
	// returning query results containing table joins
	for _, column := range columns {
		columnsPrepended = append(
			columnsPrepended,
			fmt.Sprintf("%s\x00%s", table, column),
		)
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	for i := range columnTypes {
		dataType := columnTypes[i].DatabaseTypeName()
		nativeType := b.ConvertDBSpecificDataType(dataType)
		dataTypes = append(dataTypes, nativeType)
		columnNames = append(columnNames, columnsPrepended[i])
	}
	return &TableSchema{columnNames, dataTypes}, nil
}

func getTableSchemaWithRelationships(b *Backend, query, table string) (*TableSchema, error) {
	var dataTypes []interface{}
	var columnNames []string
	var columnsPrepended []string
	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, table)
	fields := util.TypeDescriptorRelationships(td)
	currentColumnsIdx := len(b.TableSchemas[table].ColumnNames)
	currentColumns := columns[0:currentColumnsIdx]
	for _, cc := range currentColumns {
		columnsPrepended = append(
			columnsPrepended,
			fmt.Sprintf("%s\x00%s", table, cc),
		)
	}
	var previousTableIdx = currentColumnsIdx
	var newTableIdx = 0
	for _, field := range fields {
		thisTable := field.Relationship.WithTable
		newTableIdx = previousTableIdx + len(b.TableSchemas[thisTable].ColumnNames)
		currentColumns := columns[previousTableIdx:newTableIdx]
		for _, cc := range currentColumns {
			columnsPrepended = append(
				columnsPrepended,
				fmt.Sprintf("%s\x00%s", thisTable, cc),
			)
		}
		previousTableIdx = newTableIdx
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	for i := range columnTypes {
		dataType := columnTypes[i].DatabaseTypeName()
		nativeType := b.ConvertDBSpecificDataType(dataType)
		dataTypes = append(dataTypes, nativeType)
		columnNames = append(columnNames, columnsPrepended[i])
	}
	return &TableSchema{columnNames, dataTypes}, nil
}

func (b *Backend) getColumnNamesAndDataTypesFromDBTableSchema(table string) (columnNames []string, dataTypes []interface{}) {
	// Use the TableSchema containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	if util.TableHasRelationships(config.Options, table) {
		columnNames = b.TableSchemas[fmt.Sprintf("%s_relationships", table)].ColumnNames
		dataTypes = b.TableSchemas[fmt.Sprintf("%s_relationships", table)].DataTypes
		return
	}
	columnNames = b.TableSchemas[table].ColumnNames
	dataTypes = b.TableSchemas[table].DataTypes
	return
}

func (b *Backend) getColumnNamesAndDataTypesForOptionRoutes(table, columnAsOptionName, uniqueIDColumn string) (columnNames []string, dataTypes []interface{}) {
	columnNamesAndDataTypes := make(map[string]interface{})
	for i, columnName := range b.TableSchemas[table].ColumnNames {
		columnNamesAndDataTypes[columnName] = b.TableSchemas[table].DataTypes[i]
	}
	columnIDAndName := []string{
		fmt.Sprintf("%s\x00%s", table, "id"),
		fmt.Sprintf("%s\x00%s", table, "name"),
	}
	dataTypesForIDandName := []interface{}{
		columnNamesAndDataTypes[fmt.Sprintf("%s\x00%s", table, uniqueIDColumn)],
		columnNamesAndDataTypes[fmt.Sprintf("%s\x00%s", table, columnAsOptionName)],
	}
	return columnIDAndName, dataTypesForIDandName
}

func (b *Backend) queryContext(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	table := ctx.Value(util.ContextKey("table")).(string)
	columnNames, dataTypes := b.getColumnNamesAndDataTypesFromDBTableSchema(table)
	rows, err := b.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err = rowsToResults(rows, columnNames, dataTypes)
	if err != nil {
		return nil, err
	}
	return
}

func (b *Backend) queryContextForOptionRoutes(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	table := ctx.Value(util.ContextKey("table")).(string)
	columnAsOptionName := ctx.Value(util.ContextKey("columnAsOptionName")).(string)
	uniqueIDColumn := ctx.Value(util.ContextKey("uniqueIDColumn")).(string)
	columnNames, dataTypes := b.getColumnNamesAndDataTypesForOptionRoutes(table, columnAsOptionName, uniqueIDColumn)
	rows, err := b.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err = rowsToResults(rows, columnNames, dataTypes)
	if err != nil {
		return nil, err
	}
	return
}
func (h handler) interpolateQueryTemplate() (interpolatedQueryTemplate string, err error) {
	queryTemplate, err := template.New("dbquery").Parse(h.vars[0])
	if err != nil {
		return "", err
	}
	query := bytes.NewBufferString("")
	err = queryTemplate.Execute(query, h.templateData)
	if err != nil {
		return "", err
	}
	return query.String(), nil
}

func (h handler) interpolateExecTemplates(ctx context.Context, requestData map[string]interface{}) (interpolatedQuery string, args []interface{}, err error) {
	templateText := h.vars[0]
	funcMap := template.FuncMap{
		"add2": func(x int) int {
			return x + 2
		},
		"lenPlus1": func(x []string) int {
			return len(x) + 1
		},
		"head": func(x []string) string {
			return x[0]
		},
		"tail": func(x []string) []string {
			return x[1:]
		},
	}
	queryTemplate, err := template.New("dbquery").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", nil, err
	}
	query := bytes.NewBufferString("")
	err = queryTemplate.Execute(query, h.templateData)
	if err != nil {
		return "", nil, err
	}
	args = buildExecQueryArgs(ctx, requestData)
	log.When(config.Options.Logging).Infof(
		"[handler <- db] buildExecQueryArgsWithID(): returned following args:\n%s\n",
		args,
	)
	return query.String(), args, nil

}

func RunTests(t *testing.T, args ...interface{}) {
	t.Run("GetSingle", func(t *testing.T) {
		for _, tc := range TestCasesGetSingle {
			Run(t, tc, args...)
		}
	})
	t.Run("GetSingleAsOption", func(t *testing.T) {
		for _, tc := range TestCasesGetSingleAsOption {
			Run(t, tc, args...)
		}
	})
	t.Run("GetCollection", func(t *testing.T) {
		for _, tc := range TestCasesGetCollection {
			Run(t, tc, args...)
		}
	})
	t.Run("GetCollectionAsOptions", func(t *testing.T) {
		for _, tc := range TestCasesGetCollectionAsOptions {
			Run(t, tc, args...)
		}
	})
	t.Run("GetCollectionAsOptionsFilterable", func(t *testing.T) {
		for _, tc := range TestCasesGetCollectionAsOptionsFilterable {
			Run(t, tc, args...)
		}
	})
	t.Run("UpdateSingle", func(t *testing.T) {
		for _, tc := range TestCasesUpdateSingle {
			Run(t, tc, args...)
		}
	})
	t.Run("CreateSingle", func(t *testing.T) {
		for _, tc := range TestCasesCreateSingle {
			Run(t, tc, args...)
		}
	})
	t.Run("DeleteSingle", func(t *testing.T) {
		for _, tc := range TestCasesDeleteSingle {
			Run(t, tc, args...)
		}
	})
}
func Run(t *testing.T, tc TestCase, args ...interface{}) {
	if tc.Kind == "success" {
		tc.Run = itSucceeds
		tc.Run(t, tc, args...)
	} else if tc.Kind == "failure" {
		tc.Run = itFails
		tc.Run(t, tc, args...)
	} else {
		t.Errorf("testcase should either be success or failure kind")
	}
}
func itFails(t *testing.T, tc TestCase, args ...interface{}) {
	var backend *Backend
	var err error
	var mock sqlmock.Sqlmock
	usingMockedDB := true
	if len(args) > 0 {
		usingMockedDB = false
	}
	if usingMockedDB {
		// The config.Descriptor in config.Options needs to be mocked
		mockedDescriptorFile, err := mockDescriptorFile(tc.DescriptorFields)
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
		config.Options.Descriptor = config.ParseDescriptorFile(mockedDescriptorFile)
		backend, mock, err = setupBackendWithMockedDB()
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
		// initialize mock database
		tc.ExpectedQueries(mock, tc.ColumnNames, tc.RowsAsCsv)
		// mock the database table schema
		backend.TableSchemas["equipment"] = tc.TableSchema
	} else {
		driver := args[0]
		url := args[1]
		setupBackendFn := args[2].(func() *Backend)
		backend = setupBackendFn()
		err = backend.Open(
			driver,
			url,
		)
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
	}
	ts := setupTestServer(backend)
	defer ts.Close()
	tc.Request.URL, err = url.Parse(ts.URL + tc.Request.URL.String())
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	tc.Request.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(tc.Request)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found, instead we received: %v", res.StatusCode)
	}
	if string(got[:]) != tc.ExpectedResults {
		t.Errorf("Response doesn't match hat we expected\nResponse:\n%s\nExpected:\n%s\n",
			got, tc.ExpectedResults)
	}
	if usingMockedDB {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}

	}
}
func itSucceeds(t *testing.T, tc TestCase, args ...interface{}) {
	var backend *Backend
	var err error
	var mock sqlmock.Sqlmock
	usingMockedDB := true
	if len(args) > 0 {
		usingMockedDB = false
	}
	if usingMockedDB {
		// The config.Descriptor in config.Options needs to be mocked
		mockedDescriptorFile, err := mockDescriptorFile(tc.DescriptorFields)
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
		config.Options.Descriptor = config.ParseDescriptorFile(mockedDescriptorFile)
		backend, mock, err = setupBackendWithMockedDB()
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
		// initialize mock database
		tc.ExpectedQueries(mock, tc.ColumnNames, tc.RowsAsCsv)
		// mock the database table schema
		backend.TableSchemas["equipment"] = tc.TableSchema
	} else {
		driver := args[0]
		url := args[1]
		setupBackendFn := args[2].(func() *Backend)
		backend = setupBackendFn()
		err = backend.Open(
			driver,
			url,
		)
		if err != nil {
			t.Errorf("Expected no error, instead we received: %s", err)
		}
	}
	ts := setupTestServer(backend)
	fmt.Printf("back templ: %#+v\n", ts)
	defer ts.Close()
	tc.Request.URL, err = url.Parse(ts.URL + tc.Request.URL.String())
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	tc.Request.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(tc.Request)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	if strings.HasPrefix(string(res.StatusCode), "2") {
		t.Errorf("Expected HTTP 2xx, instead we received: %d", res.StatusCode)
	}
	if string(got[:]) != tc.ExpectedResults {
		t.Errorf("Response doesn't match what we expected\nResponse:\n%q\nExpected:\n%q\n",
			got, tc.ExpectedResults)
	}
	if usingMockedDB {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}

	}
}
func setupBackendWithMockedDB() (b *Backend, mock sqlmock.Sqlmock, err error) {
	b = NewBackend()
	b.Templates = queryTemplates
	b.DB, mock, err = sqlmock.New()
	if err != nil {
		return nil, mock, fmt.Errorf(
			"an error '%s' was not expected when opening a stub database connection",
			err,
		)
	}
	return
}
func setupTestServer(b *Backend) *httptest.Server {
	router := b.GetHandler().(*mux.Router)
	ts := httptest.NewUnstartedServer(router)
	router.Use(middleware.BasicAuth)
	router.Use(middleware.RequestInjector)
	router.Use(middleware.ResponseInjector)
	server := &http.Server{}
	server.Handler = router
	ts.Config = server
	ts.Start()
	return ts
}
func mockDescriptorFile(testCaseDescriptorFields string) (io.Reader, error) {
	mockedDescriptorFile := fmt.Sprintf(
		descriptorFileBase,
		testCaseDescriptorFields,
	)
	return strings.NewReader(mockedDescriptorFile), nil
}
