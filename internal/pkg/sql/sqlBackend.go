// Package sql defines a Backend that is responsible for communicating
// with SQL databases and other external systems
package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/signavio/workflow-connector/internal/app/backend"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/filter"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	ErrUnexpectedJSON         = errors.New("Received JSON data that we are unable to parse")
	ErrMismatchedAffectedRows = errors.New("The amount of rows affected should be sane")
	ErrNoLastInsertID         = errors.New("Database does not support getting the last inserted ID")
	filterPredicateMapping    = func() map[filter.Predicate]string {
		return map[filter.Predicate]string{
			filter.Equal: "=",
		}
	}
	coerceArgDateTimeFunc = func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool) {
		dateTimeWorkflowFormat := `2006-01-02T15:04:05.000Z`
		if result, ok := requestData[field.Key]; ok {
			if result != nil {
				stringifiedDateTime := result.(string)
				parsedDateTime, err := time.ParseInLocation(
					dateTimeWorkflowFormat, stringifiedDateTime, time.UTC,
				)
				if err != nil {
					log.When(config.Options.Logging).Infof(
						"[backend] error when trying to coerce arg of type 'datetime': %s\n",
						err,
					)
					return nil, ok
				}
				return parsedDateTime, ok
			}
			return result, ok
		}
		return
	}
	coerceArgFuncs = map[string]func(map[string]interface{}, *descriptor.Field) (interface{}, bool){
		"default": func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool) {
			result, ok = requestData[field.Key]
			return
		},
		"money": func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool) {
			if result, ok = requestData[field.Type.Amount.Key]; ok {
				return result, ok
			}
			if result, ok := requestData[field.Type.Currency.Key]; ok {
				return result, ok
			}
			return
		},
		"datetime": coerceArgDateTimeFunc,
		"date":     coerceArgDateTimeFunc,
		"time":     coerceArgDateTimeFunc,
	}
)

type SqlBackend struct {
	*backend.Backend
	DB                     *sql.DB
	Templates              map[string]string
	SchemaMapping          map[string]*descriptor.SchemaMapping
	FilterPredicateMapping map[filter.Predicate]string
	NewSchemaMapping       func([]string, []*sql.ColumnType) (*descriptor.SchemaMapping, error)
	Transactions           sync.Map
}

func New() endpoint.Endpoint {
	s := &SqlBackend{}
	s.Backend = &backend.Backend{}
	s.GetSchemaMappingFunc = s.getSchemaMapping
	s.GetQueryTemplateFunc = s.getQueryTemplate
	s.CoerceArgFuncs = coerceArgFuncs
	s.OpenFunc = s.open
	s.CommitTxFunc = s.commitTx
	s.CreateTxFunc = s.createTx
	s.QueryContextFunc = s.queryContext
	s.ExecContextFunc = s.execContext
	s.SchemaMapping = make(map[string]*descriptor.SchemaMapping)
	s.FilterPredicateMapping = filterPredicateMapping()
	s.GetFilterPredicateMappingFunc = s.getFilterPredicateMapping
	s.NewSchemaMapping = s.newSchemaMapping
	return s
}

func (s *SqlBackend) open(args ...interface{}) error {
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		config.Options.Database.Driver,
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
	s.DB = db
	err = s.SaveSchemaMapping()
	if err != nil {
		return fmt.Errorf("Error saving table schema: %s", err)
	}
	return nil
}

func (s *SqlBackend) SaveSchemaMapping() (err error) {
	log.When(config.Options.Logging).Infoln("[backend] query database and save table schemas")
	for _, table := range config.Options.Database.Tables {
		// table has no relationships defined in descriptor.json
		log.When(config.Options.Logging).Infof(
			"[backend] schema for table %v:\n",
			table.Name,
		)
		err := s.populateBackendSchemaMapping(table.Name, "GetTableSchema")
		if err != nil {
			return err
		}
		log.When(config.Options.Logging).Infof("%#+v\n", s.SchemaMapping)
	}
	for _, table := range config.Options.Database.Tables {
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
			err := s.addRelationshipsToBackendSchemaMapping(
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
		s.SchemaMapping,
	)
	return nil
}
func (s *SqlBackend) getQueryTemplate(name string) string {
	return s.Templates[name]
}
func (s *SqlBackend) getSchemaMapping(typeDescriptor string) *descriptor.SchemaMapping {
	return s.SchemaMapping[typeDescriptor]
}
func (s *SqlBackend) getFilterPredicateMapping(predicate filter.Predicate) string {
	return s.FilterPredicateMapping[predicate]
}
func (s *SqlBackend) populateBackendSchemaMapping(tableName, templateName string) error {
	template := s.getQueryTemplate(templateName)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{template},
		TemplateData: struct {
			TableName string
		}{
			TableName: tableName,
		},
		CoerceArgFuncs: s.GetCoerceArgFuncs(),
	}
	query, _, err := queryTemplate.Interpolate(
		context.WithValue(context.Background(), util.ContextKey("table"), tableName),
		nil,
	)
	if err != nil {
		return err
	}
	tableSchema, err := s.retrieveSchemaMapping(query, tableName)
	if err != nil {
		return fmt.Errorf("Unable to retrieve columns and data types from table schema: %s", err)
	}
	s.SchemaMapping[tableName] = tableSchema
	return nil
}

func (s *SqlBackend) addRelationshipsToBackendSchemaMapping(tableName, templateName, uniqueIDColumn string, relationships []*descriptor.Field) error {
	template := s.getQueryTemplate(templateName)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{template},
		TemplateData: struct {
			TableName      string
			Relations      []*descriptor.Field
			UniqueIDColumn string
		}{
			TableName:      tableName,
			Relations:      relationships,
			UniqueIDColumn: uniqueIDColumn,
		},
		CoerceArgFuncs: s.GetCoerceArgFuncs(),
	}
	query, _, err := queryTemplate.Interpolate(
		context.WithValue(context.Background(), util.ContextKey("table"), tableName),
		nil,
	)
	if err != nil {
		return err

	}
	tableSchema, err := s.retrieveSchemaMappingWithRelationships(query, tableName)
	if err != nil {
		return fmt.Errorf("Unable to retrieve columns and data types from table schema: %s", err)
	}
	s.SchemaMapping[fmt.Sprintf("%s\x00relationships", tableName)] = tableSchema
	return nil
}

func (s *SqlBackend) retrieveSchemaMapping(query, table string) (*descriptor.SchemaMapping, error) {
	log.When(config.Options.Logging).Infoln(query)

	rows, err := s.DB.Query(query)
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
	return s.NewSchemaMapping(columnsPrepended, columnTypes)
}

func (s *SqlBackend) retrieveSchemaMappingWithRelationships(query, table string) (*descriptor.SchemaMapping, error) {
	log.When(config.Options.Logging).Infoln(query)

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnsPrepended := prependTableNameToColumnsInJoinedTables(s, table, columns)
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	if len(columnTypes) == 0 {
		return nil, fmt.Errorf(
			"unable to get the data types of database table columns",
		)
	}
	return s.NewSchemaMapping(columnsPrepended, columnTypes)
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
func prependTableNameToColumnsInJoinedTables(s *SqlBackend, table string, columns []string) []string {
	var columnsPrepended []string
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, table)
	fields := util.TypeDescriptorRelationships(td)
	currentColumnsIdx := len(s.SchemaMapping[table].FieldNames)
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

		newTableIdx = previousTableIdx + len(s.SchemaMapping[thisTable].FieldNames)
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

func (s *SqlBackend) newSchemaMapping(columnsWithTable []string, columnTypes []*sql.ColumnType) (*descriptor.SchemaMapping, error) {
	var backendTypes, golangTypes, workflowTypes []interface{}
	var fieldNames []string
	for i := range columnTypes {
		backendType := columnTypes[i].DatabaseTypeName()
		if backendType == "" {
			return nil, fmt.Errorf(
				"unable to get the type in use by the backend",
			)
		}
		golangType := s.CastBackendTypeToGolangType(backendType)
		if golangType == nil {
			return nil, fmt.Errorf(
				"unable to get the native golang type",
			)
		}
		workflowType := GetWorkflowType(columnsWithTable[i])
		if workflowType == nil {
			return nil, fmt.Errorf(
				"unable to get the workflow type specified in descriptor.json",
			)
		}
		workflowTypes = append(workflowTypes, workflowType)
		backendTypes = append(backendTypes, backendType)
		golangTypes = append(golangTypes, golangType)
		fieldNames = append(fieldNames, columnsWithTable[i])
	}
	return &descriptor.SchemaMapping{fieldNames, backendTypes, golangTypes, workflowTypes}, nil
}

func GetWorkflowType(columnWithTable string) interface{} {
	var tableName string
	tableNamePrefix := strings.IndexRune(columnWithTable, '\x00')
	tableName = columnWithTable[0:tableNamePrefix]
	typeDescriptor := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		tableName,
	)
	for _, field := range typeDescriptor.Fields {
		if field.Type.Name == "date" {
			return field.Type.Kind
		}
		return field.Type.Name
	}
	return nil
}

func (s *SqlBackend) getColumnNamesAndDataTypesFromDBSchemaMapping(table string, withRelationships bool) (fieldNames []string, golangTypes []interface{}) {
	// Use the SchemaMapping containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	if withRelationships {
		fieldNames = s.GetSchemaMapping(fmt.Sprintf("%s\x00relationships", table)).FieldNames
		golangTypes = s.GetSchemaMapping(fmt.Sprintf("%s\x00relationships", table)).GolangTypes
		return
	}
	fieldNames = s.GetSchemaMapping(table).FieldNames
	golangTypes = s.GetSchemaMapping(table).GolangTypes
	return
}

func (s *SqlBackend) getColumnNamesAndDataTypesForOptionRoutes(table, columnAsOptionName, uniqueIDColumn string) (columnNames []string, dataTypes []interface{}) {
	columnNamesAndDataTypes := make(map[string]interface{})
	for i, columnName := range s.GetSchemaMapping(table).FieldNames {
		columnNamesAndDataTypes[columnName] = s.GetSchemaMapping(table).GolangTypes[i]
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

func (s *SqlBackend) LoadTx(key interface{}) (value interface{}, ok bool) {
	return s.Transactions.Load(key)
}

func (s *SqlBackend) StoreTx(key, value interface{}) {
	s.Transactions.Store(key, value)
	return
}

func (s *SqlBackend) DeleteTx(key interface{}) {
	s.Transactions.Delete(key)
	return
}
