// Package sql defines a Backend that is responsible for communicating
// with SQL databases
package sql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/backend/sql/mysql"
	"github.com/signavio/workflow-connector/internal/pkg/backend/sql/postgres"
	"github.com/signavio/workflow-connector/internal/pkg/backend/sql/sqlite"
	"github.com/signavio/workflow-connector/internal/pkg/backend/sql/sqlserver"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
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
)

type Backend struct {
	ConvertDBSpecificDataType func(string) interface{}
	DB                        *sql.DB
	TableSchemas              map[string]*TableSchema
	Templates                 map[string]string
	Transactions              sync.Map
	TransactDirectly          func(context.Context, *sql.DB, string, ...interface{}) (sql.Result, error)
	TransactWithinTx          func(context.Context, *sql.Tx, string, ...interface{}) (sql.Result, error)
}

// TableSchema stores the schema of the database in use
type TableSchema struct {
	ColumnNames []string
	DataTypes   []interface{}
}

type handler struct {
	vars         []string
	templateData interface{}
}

// NewBackend will return a backend with database specific templates
func NewBackend(driver string) (b *Backend) {
	b = &Backend{}
	switch driver {
	case "sqlserver":
		b.TableSchemas = make(map[string]*TableSchema)
		b.ConvertDBSpecificDataType = sqlserver.ConvertFromSqlserverDataType
		b.Templates = sqlserver.QueryTemplates
		return
	case "sqlite":
		b.TableSchemas = make(map[string]*TableSchema)
		b.ConvertDBSpecificDataType = sqlite.ConvertFromSqliteDataType
		b.Templates = sqlite.QueryTemplates
		return
	case "mysql":
		b.TableSchemas = make(map[string]*TableSchema)
		b.ConvertDBSpecificDataType = mysql.ConvertFromMysqlDataType
		b.Templates = mysql.QueryTemplates
		return
	case "postgres":
		b.TableSchemas = make(map[string]*TableSchema)
		b.ConvertDBSpecificDataType = postgres.ConvertFromPostgresDataType
		b.Templates = postgres.QueryTemplates
		b.TransactDirectly = postgres.ExecContextDirectly
		b.TransactWithinTx = postgres.ExecContextWithinTx
		return
	case "sqlmock":
		// When using a sqlmock just return an empty backend
		// with initialized TableSchemas
		b.TableSchemas = make(map[string]*TableSchema)
		return
	}
	return
}

// Open a connection to the backend database
func (b *Backend) Open(args ...interface{}) error {
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		config.Options.Endpoint.Driver,
	)
	driver := args[0].(string)
	url := args[1].(string)
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		driver,
	)
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
	for _, table := range config.Options.Endpoint.Tables {
		// table has no relationships defined in descriptor.json
		log.When(config.Options.Logging).Infof(
			"[backend] schema for table %v:\n",
			table.Name,
		)
		err := b.populateBackendTableSchemas(table.Name, "GetTableSchema")
		if err != nil {
			return err
		}
		log.When(config.Options.Logging).Infof("%#+v\n", b.TableSchemas)
	}
	for _, table := range config.Options.Endpoint.Tables {
		if util.TableHasRelationships(config.Options, table.Name) {
			log.When(config.Options.Logging).Infof(
				"[backend] schema for table %v:\n",
				table.Name+" (with relationships)",
			)

			td := util.GetTypeDescriptorUsingDBTableName(
				config.Options.Descriptor.TypeDescriptors,
				table.Name,
			)
			tdRelationships := util.TypeDescriptorRelationships(td)
			tdUniqueIDColumn := td.UniqueIdColumn
			err := b.addRelationshipsToBackendTableSchemas(
				table.Name,
				"GetTableWithRelationshipsSchema",
				tdUniqueIDColumn,
				tdRelationships,
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

func (b *Backend) populateBackendTableSchemas(tableName, templateName string) error {
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
	log.When(config.Options.Logging).Infof("%+v\n", query)
	log.When(config.Options.Logging).Infof("%+v\n", templateName)

	tableSchema, err := b.getTableSchema(query, tableName)
	if err != nil {
		return fmt.Errorf("Unable to retrieve columns and data types from table schema: %s", err)
	}
	b.TableSchemas[tableName] = tableSchema
	return nil
}

func (b *Backend) addRelationshipsToBackendTableSchemas(tableName, templateName, uniqueIDColumn string, relationships []*config.Field) error {
	queryTemplate := b.Templates[templateName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			Relations      []*config.Field
			UniqueIDColumn string
		}{
			TableName:      tableName,
			Relations:      relationships,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	query, err := handler.interpolateQueryTemplate()
	if err != nil {
		return err

	}
	tableSchema, err := b.getTableSchemaWithRelationships(query, tableName)
	if err != nil {
		return fmt.Errorf("Unable to retrieve columns and data types from table schema: %s", err)
	}
	b.TableSchemas[fmt.Sprintf("%s\x00relationships", tableName)] = tableSchema
	return nil
}

func (b *Backend) getTableSchema(query, table string) (*TableSchema, error) {
	log.When(config.Options.Logging).Infoln(query)

	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf(
			"unable to get a list of columns from the database table",
		)
	}
	columnsPrepended := prependTableNameToColumns(table, columns)
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	if len(columnTypes) == 0 {
		return nil, fmt.Errorf(
			"unable to get the data types of database table columns",
		)
	}
	return b.newTableSchema(columnsPrepended, columnTypes)
}

func (b *Backend) getTableSchemaWithRelationships(query, table string) (*TableSchema, error) {
	log.When(config.Options.Logging).Infoln(query)

	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnsPrepended := prependTableNameToColumnsInJoinedTables(b, table, columns)
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	if len(columnTypes) == 0 {
		return nil, fmt.Errorf(
			"unable to get the data types of database table columns",
		)
	}
	return b.newTableSchema(columnsPrepended, columnTypes)
}

// prependTablenameToColumns will prepend the table name to each column name
// since this makes it easier to keep track of which column belongs to
// which table when returning query results containing table joins
func prependTableNameToColumns(table string, columns []string) []string {
	var columnsPrepended []string
	for _, column := range columns {
		columnsPrepended = append(
			columnsPrepended,
			fmt.Sprintf("%s\x00%s", table, column),
		)
	}
	return columnsPrepended
}

// prependTablenameToColumnsInJoinedTables will prepend the table name
// to each column name on a query result that contains multiple tables
func prependTableNameToColumnsInJoinedTables(b *Backend, table string, columns []string) []string {
	var columnsPrepended []string
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
	return columnsPrepended
}

func (b *Backend) newTableSchema(columnWithTable []string, columnTypes []*sql.ColumnType) (*TableSchema, error) {
	var dataTypes []interface{}
	var columnNames []string
	for i := range columnTypes {
		dataType := columnTypes[i].DatabaseTypeName()
		if dataType == "" {
			return nil, fmt.Errorf(
				"unable to get the data types of database table columns",
			)
		}
		nativeType := b.ConvertDBSpecificDataType(dataType)
		if nativeType == "" {
			return nil, fmt.Errorf(
				"unable to get the data types of database table columns",
			)
		}
		dataTypes = append(dataTypes, nativeType)
		columnNames = append(columnNames, columnWithTable[i])
	}
	return &TableSchema{columnNames, dataTypes}, nil
}

func (b *Backend) getColumnNamesAndDataTypesFromDBTableSchema(table string, withRelationships bool) (columnNames []string, dataTypes []interface{}) {
	// Use the TableSchema containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	if withRelationships {
		columnNames = b.TableSchemas[fmt.Sprintf("%s\x00relationships", table)].ColumnNames
		dataTypes = b.TableSchemas[fmt.Sprintf("%s\x00relationships", table)].DataTypes
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
	relationships := ctx.Value(util.ContextKey("relationships")).([]*config.Field)
	currentRoute := ctx.Value(util.ContextKey("currentRoute")).(string)
	var columnNames []string
	var dataTypes []interface{}
	// Use the TableSchema containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	// and the current route is GetSingle
	if relationships != nil && currentRoute == "GetSingle" {
		columnNames = b.TableSchemas[fmt.Sprintf("%s\x00relationships", table)].ColumnNames
		dataTypes = b.TableSchemas[fmt.Sprintf("%s\x00relationships", table)].DataTypes
	} else {
		columnNames = b.TableSchemas[table].ColumnNames
		dataTypes = b.TableSchemas[table].DataTypes
	}
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
