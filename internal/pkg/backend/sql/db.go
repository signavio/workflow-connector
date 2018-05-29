package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

func (b *Backend) execContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	requestTx := ctx.Value(util.ContextKey("tx")).(string)
	if requestTx == "" {
		// User has not specified an existing transaction to execute within
		tx, err := b.DB.Begin()
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
		txi, _ := b.Transactions.Load(requestTx)
		tx := txi.(*sql.Tx)
		result, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		tx.Rollback()
		return
	}
	result, err = b.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return
}

func buildExecQueryArgs(ctx context.Context, requestData map[string]interface{}) (args []interface{}) {
	currentTable := ctx.Value(util.ContextKey("table")).(string)
	id := ctx.Value(util.ContextKey("id")).(string)
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, currentTable)
	var val interface{}
	var ok bool
	appendRequestDataToArgs := func(args []interface{}, val interface{}) []interface{} {
		switch v := val.(type) {
		case string:
			return append(args, v)
		case bool:
			return append(args, v)
		case float64:
			return append(args, v)
		case nil:
			return append(args, v)
		}
		return []interface{}{}
	}
	for _, field := range td.Fields {
		if field.Type.Name == "money" {
			if val, ok = requestData[field.Type.Amount.Key]; ok {
				args = appendRequestDataToArgs(args, val)
			}
			if val, ok = requestData[field.Type.Currency.Key]; ok {
				args = appendRequestDataToArgs(args, val)
			}
		} else {
			if val, ok = requestData[field.Key]; ok {
				args = appendRequestDataToArgs(args, val)
			}
		}
	}
	if id != "" {
		args = append(args, id)
	}
	return
}

func deduplicateSingleResource(data []interface{}, td *config.TypeDescriptor) []interface{} {
	fields := util.TypeDescriptorRelationships(td)
	for _, field := range fields {
		var fieldResultSet []map[string]interface{}
		var relationshipTableContainsResults = false
		for _, datum := range data {
			if tableResults, ok := datum.(map[string]interface{})[field.Relationship.WithTable].(map[string]interface{}); ok {
				// If the result set of a related table is empty, then all values
				// will be equal to nil, so do not append it to the fieldResultSet
				for _, value := range tableResults {
					if value != nil {
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

func queryContext(ctx context.Context, b *Backend, param string, queryText string) (rows *sql.Rows, err error) {
	if len(param) == 0 {
		rows, err = b.DB.QueryContext(ctx, queryText)
		if err != nil {
			return nil, err
		}
		return
	}
	rows, err = b.DB.QueryContext(ctx, queryText, param)
	if err != nil {
		return nil, err
	}
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
		tableNamePrefix := strings.IndexRune(columns[i], '_')
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
	case *NullTime:
		if v.Valid {
			tableResult[columnName] = v.Time
			result[tableName] = tableResult
		} else {
			tableResult[columnName] = nil
			result[tableName] = tableResult
		}
	}
	// Signavio Workflow Accelerator Connector API
	// requires an id field to be of type string
	switch v := result[tableName].(map[string]interface{})["id"].(type) {
	case int64:
		tableResult["id"] = fmt.Sprintf("%d", v)
		result[tableName] = tableResult
	case time.Time:
		tableResult["id"] = v.String()
		result[tableName] = tableResult
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
