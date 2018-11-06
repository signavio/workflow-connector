package postgres

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/lib/pq"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type lastId struct {
	id int64
}

var (
	QueryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			"   = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = $1",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE id = $1",
		"GetCollection": "SELECT * " +
			"FROM {{.TableName}} " +
			"ORDER BY {{.UniqueIDColumn}} ASC",
		"GetCollectionFilterable": "SELECT * " +
			"FROM {{.TableName}} " +
			"WHERE {{.FilterOnColumn}} {{.Operator}} $1",
		"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"ORDER BY {{.UniqueIDColumn}} ASC",
		"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE CAST ({{.ColumnAsOptionName}} AS TEXT) ILIKE $1",
		"UpdateSingle": "UPDATE {{.TableName}} " +
			"SET {{.ColumnNames | head}} = $1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  {{$element}} = ${{(add2 $index)}}" +
			"{{end}} " +
			"WHERE {{.UniqueIDColumn}}= ${{(lenPlus1 .ColumnNames)}}",
		"CreateSingle": "INSERT INTO {{.TableName}}" +
			"({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}," +
			"  {{.}}" +
			"{{end}}) " +
			"VALUES($1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  ${{$index | add2}}" +
			"{{end}}) " +
			"RETURNING {{.UniqueIDColumn}}",
		"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = $1",
		"GetTableSchema": "SELECT * " +
			"FROM {{.TableName}} " +
			"LIMIT 1",
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			" = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}{{end}} LIMIT 1",
	}
	integer = []string{
		"INT2",
		"INT4",
		"INT8",
	}
	text = []string{
		"CHAR",
		"VARCHAR",
		"TEXT",
		"BYTEA",
	}
	numeric = []string{
		"NUMERIC",
		"MONEY",
	}
	dateTime = []string{
		"TIMESTAMP",
		"TIMESTAMPTZ",
		"DATE",
		"TIME",
		"TIMETZ",
	}
	boolean = []string{
		"BOOL",
	}
)

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return 0, nil
}

func ExecContextDirectly(ctx context.Context, db *sql.DB, query string, args ...interface{}) (result sql.Result, err error) {
	var id int64
	if err = db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return nil, err
	}
	result = &lastId{id}
	return result, nil
}
func ExecContextWithinTx(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (result sql.Result, err error) {
	var id int64
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return nil, err
	}
	result = &lastId{id}
	return result, nil
}
func ConvertFromPostgresDataType(fieldDataType string) interface{} {
	switch {
	case isOfDataType(integer, fieldDataType):
		return &sql.NullInt64{}
	case isOfDataType(text, fieldDataType):
		return &sql.NullString{}
	case isOfDataType(numeric, fieldDataType):
		return &sql.NullFloat64{}
	case isOfDataType(dateTime, fieldDataType):
		return &util.NullTime{}
	case isOfDataType(boolean, fieldDataType):
		return &sql.NullBool{}
	default:
		return &sql.NullString{}
	}
}

func isOfDataType(ts []string, fieldDataType string) (result bool) {
	result = false
	for _, t := range ts {
		if strings.HasPrefix(strings.ToUpper(fieldDataType), t) {
			return true
		}
	}
	return
}
