package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sdaros/workflow-db-connector/pkg/config"
	"github.com/sdaros/workflow-db-connector/pkg/util"
)

func (r *getSingle) getQueryResults(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	results, err = getQueryResults(ctx, r.backend.DB, query, r.columnNames, r.dataTypes, args...)
	if util.TableHasRelationships(r.backend.Cfg, r.table) {
		return deduplicateSingleResource(
				results,
				util.TypeDescriptorForCurrentTable(r.backend.Cfg.Descriptor.TypeDescriptors, r.table)),
			nil
	}
	return
}
func (r *getCollection) getQueryResults(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	return getQueryResults(ctx, r.backend.DB, query, r.columnNames, r.dataTypes, args...)
}

func (r *getSingleAsOption) getQueryResults(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	return getQueryResults(ctx, r.backend.DB, query, r.columnNames, r.dataTypes, args...)
}

func (r *getCollectionAsOptions) getQueryResults(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	return getQueryResults(ctx, r.backend.DB, query, r.columnNames, r.dataTypes, args...)
}

func (r *getCollectionAsOptionsFilterable) getQueryResults(ctx context.Context, query string, args ...interface{}) (results []interface{}, err error) {
	return getQueryResults(ctx, r.backend.DB, query, r.columnNames, r.dataTypes, args...)
}

func getQueryResults(ctx context.Context, db *sql.DB, query string, columnNames []string, dataTypes []interface{}, args ...interface{}) (results []interface{}, err error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err = rowsToResults(rows, columnNames, dataTypes)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return
	}
	return
}
func deduplicateSingleResource(data []interface{}, td *config.TypeDescriptor) []interface{} {
	fields := util.TypeDescriptorRelationships(td)
	for _, field := range fields {
		var fieldResultSet []map[string]interface{}
		var relationshipTableContainsResults = false
		for _, datum := range data {
			tableResults := datum.(map[string]interface{})[field.Relationship.WithTable].(map[string]interface{})
			// If the result set of a related table is empty, then all values
			// will be equal to nil, so do not apend it to the fieldResultSet
			for _, value := range tableResults {
				if value != nil {
					relationshipTableContainsResults = true
				}
			}
			if relationshipTableContainsResults {
				fieldResultSet = util.AppendNoDuplicates(fieldResultSet, tableResults)
			}
		}
		data[0].(map[string]interface{})[td.TableName].(map[string]interface{})[field.Key] = map[string]interface{}{
			field.Relationship.WithTable: fieldResultSet,
		}
	}
	return append([]interface{}{}, data[0])
}

func (b *Backend) execContext(ctx context.Context, query string, args []interface{}) (results []interface{}, err error) {
	tx, err := b.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			tx.Rollback()
		}
	}()
	result, err := b.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affectedRows != 1 {
		return nil, ErrMismatchedAffectedRows
	}
	return
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

func (b *Backend) buildExecQueryArgs(ctx context.Context) (args []interface{}) {
	currentTable := ctx.Value(config.ContextKey("table")).(string)
	for _, column := range b.Cfg.TableSchemas[currentTable].ColumnNames {
		// Remove tablename prefix
		tableNamePrefix := strings.IndexRune(column, '_')
		columnName := column[tableNamePrefix+1 : len(column)]
		if val, ok := b.RequestData[columnName]; ok {
			switch v := val.(type) {
			case string:
				args = append(args, v)
			case bool:
				args = append(args, v)
			case float64:
				args = append(args, v)
			case nil:
				args = append(args, v)
			}
		}
	}
	return
}

func (b *Backend) buildExecQueryArgsWithID(ctx context.Context, id string) (args []interface{}) {
	args = b.buildExecQueryArgs(ctx)
	args = append(args, id)
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
