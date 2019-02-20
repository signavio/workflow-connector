package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

func (s *SqlBackend) execContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	requestTx := ctx.Value(util.ContextKey("tx")).(string)
	if requestTx == "" {
		// User has not specified an existing transaction to execute within.
		// However, we will still run the exec statement within a new
		// transaction
		tx, err := s.DB.Begin()
		if err != nil {
			return nil, err
		}
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p) // re-throw panic after tx.Rollback()
			} else if err != nil {
				tx.Rollback()
			} else {
				err = tx.Commit()
			}
		}()
	} else {
		// We assume the transacation is a valid one
		txi, _ := s.Transactions.Load(requestTx)
		tx := txi.(*sql.Tx)
		result, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		tx.Commit()
		return
	}
	result, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return
}

func (s *SqlBackend) queryContext(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	currentRoute := ctx.Value(util.ContextKey("currentRoute")).(string)
	switch currentRoute {
	case "GetSingleAsOption", "GetCollectionAsOptions":
		return s.queryContextForOptionRoutes(ctx, query, args...)
	case "GetCollectionAsOptionsFilterable", "GetCollectionAsOptionsWithParams":
		return s.queryContextForOptionRoutes(ctx, query, args...)
	case "GetSingle":
		return s.queryContextForGetSingleRoute(ctx, query, args...)
	default:
		return s.queryContextForNonOptionRoutes(ctx, query, args...)
	}
}
func (s *SqlBackend) queryContextForGetSingleRoute(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	tableName := ctx.Value(util.ContextKey("table")).(string)
	results, err = s.queryContextForNonOptionRoutes(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		msg := &util.ResponseMessage{
			Code: http.StatusNotFound,
			Msg: fmt.Sprintf(
				"Resource with uniqueID '%s' not found in %s table",
				args[len(args)-1], tableName,
			),
		}
		return nil, msg
	}
	// deduplicate the results fetched when querying the database
	results = deduplicateSingleResource(
		results,
		util.GetTypeDescriptorUsingDBTableName(
			config.Options.Descriptor.TypeDescriptors,
			tableName,
		),
	)
	return
}
func (s *SqlBackend) queryContextForNonOptionRoutes(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	table := ctx.Value(util.ContextKey("table")).(string)
	relationships := ctx.Value(util.ContextKey("relationships")).([]*descriptor.Field)
	currentRoute := ctx.Value(util.ContextKey("currentRoute")).(string)
	var columnNames []string
	var dataTypes []interface{}
	// Use the TableSchema containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	// and the current route is GetSingle
	if relationships != nil && currentRoute == "GetSingle" {
		columnNames = s.GetSchemaMapping(fmt.Sprintf("%s\x00relationships", table)).FieldNames
		dataTypes = s.GetSchemaMapping(fmt.Sprintf("%s\x00relationships", table)).GolangTypes
	} else {
		columnNames = s.GetSchemaMapping(table).FieldNames
		dataTypes = s.GetSchemaMapping(table).GolangTypes
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
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

func (s *SqlBackend) queryContextForOptionRoutes(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	table := ctx.Value(util.ContextKey("table")).(string)
	columnAsOptionName := ctx.Value(util.ContextKey("columnAsOptionName")).(string)
	uniqueIDColumn := ctx.Value(util.ContextKey("uniqueIDColumn")).(string)
	columnNames, dataTypes := s.getColumnNamesAndDataTypesForOptionRoutes(table, columnAsOptionName, uniqueIDColumn)
	log.When(config.Options.Logging).Infof("[db] column: %s\n datatypes: %+v\n", columnNames, dataTypes)
	rows, err := s.DB.QueryContext(ctx, query, args...)
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

func deduplicateSingleResource(data []interface{}, td *descriptor.TypeDescriptor) []interface{} {
	fields := util.TypeDescriptorRelationships(td)
	for _, field := range fields {
		var fieldResultSet []map[string]interface{}
		var relationshipTableContainsResults = false
		for _, datum := range data {
			if tableResults, ok := datum.(map[string]interface{})[field.Relationship.WithTable].(map[string]interface{}); ok {
				// If the result set of a related table is empty, then all values
				// will equal nil (or the empty string for oracle db) so do not
				// append it to the fieldResultSet
				for _, value := range tableResults {
					var isEmptyValue bool
					valueString, ok := value.(string)
					if ok {
						isEmptyValue = valueString != ""
					} else {
						isEmptyValue = value != nil
					}
					if isEmptyValue {
						relationshipTableContainsResults = true
					}
				}
				if relationshipTableContainsResults {
					fieldResultSet = util.AppendNoDuplicates(fieldResultSet, tableResults)
				}
			}
		}
		data[0].(map[string]interface{})[td.TableName].(map[string]interface{})[field.Key] = map[string]interface{}{
			field.Relationship.WithTable: fieldResultSet,
		}
	}
	return append([]interface{}{}, data[0])
}

func (s *SqlBackend) createTx(timeout time.Duration) (txUUID uuid.UUID, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		return uuid.UUID{}, err
	}
	txUUID = uuid.NewV4()
	if err != nil {
		cancel()
		return uuid.UUID{}, err
	}
	s.StoreTx(fmt.Sprintf("%s", txUUID), tx)
	log.When(config.Options.Logging).Infof("[handler] added transaction %s to backend\n", txUUID)
	// Explicitly call cancel after delay
	go func(c context.CancelFunc, d time.Duration, id uuid.UUID) {
		select {
		case <-time.After(d):
			c()
			_, ok := s.Transactions.Load(fmt.Sprintf("%s", id))
			if ok {
				s.Transactions.Delete(id)
				log.When(config.Options.Logging).Infof("[handler] timeout expired: \n"+
					"transaction %s has been deleted from backend\n", id)
			}
		}
	}(cancel, timeout, txUUID)
	return
}

func (s *SqlBackend) commitTx(txUUID string) (err error) {
	tx, ok := s.LoadTx(txUUID)
	if !ok {
		return errors.New(fmt.Sprintf("%d", http.StatusNotFound))
	}
	if err := tx.(*sql.Tx).Commit(); err != nil {
		return err
	}
	s.DeleteTx(txUUID)
	return
}

func rowsToResults(rows *sql.Rows, columnNames []string, dataTypes []interface{}) (results []interface{}, err error) {
	for rows.Next() {
		result, err := processRow(rows, columnNames, dataTypes)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return
}

func processRow(rows *sql.Rows, columns []string, values []interface{}) (result map[string]interface{}, err error) {
	err = rows.Scan(values...)
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	tableResult := make(map[string]interface{})
	var previous = ""
	for i := 0; i < len(columns); i++ {
		tableNamePrefix := strings.IndexRune(columns[i], '\x00')
		var tableName, columnName string
		tableName = columns[i][0:tableNamePrefix]
		columnName = columns[i][tableNamePrefix+1 : len(columns[i])]
		if tableName != previous {
			tableResult = make(map[string]interface{})
		}
		result, previous = switchOnValueType(
			tableName, columnName, values[i], tableResult, result,
		)
	}
	return result, nil
}
func switchOnValueType(tableName, columnName string, value interface{}, tableResult, result map[string]interface{}) (map[string]interface{}, string) {
	switch v := value.(type) {
	case *sql.NullBool:
		if v.Valid {
			tableResult[columnName] = v.Bool
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	case *sql.NullString:
		if v.Valid {
			tableResult[columnName] = v.String
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	case *sql.NullInt64:
		if v.Valid {
			tableResult[columnName] = v.Int64
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	case *sql.NullFloat64:
		if v.Valid {
			tableResult[columnName] = v.Float64
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	case *util.NullTime:
		if v.Valid {
			tableResult[columnName] = v.Time
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	}
	return result, tableName
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
