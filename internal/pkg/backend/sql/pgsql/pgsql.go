package pgsql

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/backend/sql"
)

type lastId struct {
	id int64
}

var (
	queryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			"   = _{{$.TableName}}.{{.UniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = $1",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE id = $1",
		"GetCollection": "SELECT * " +
			"FROM {{.TableName}}",
		"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}}",
		"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE CAST ({{.ColumnAsOptionName}} AS TEXT LIKE $1",
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
		"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = ?",
		"GetTableSchema": "SELECT * " +
			"FROM {{.TableName}} " +
			"LIMIT 1",
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.{{.UniqueIDColumn}}{{end}} LIMIT 1",
	}
)

func NewPgsqlBackend() (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend()
	b.ConvertDBSpecificDataType = convertFromPgsqlDataType
	b.Templates = queryTemplates
	b.TransactDirectly = execContextDirectly
	b.TransactWithinTx = execContextWithinTx
	return b
}

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return 0, nil
}

func execContextDirectly(ctx context.Context, db *sql.DB, query string, args ...interface{}) (result sql.Result, err error) {
	var id int64
	if err = db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return nil, err
	}
	result = &lastId{id}
	return result, nil
}
func execContextWithinTx(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (result sql.Result, err error) {
	var id int64
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return nil, err
	}
	result = &lastId{id}
	return result, nil
}
func convertFromPgsqlDataType(fieldDataType string) interface{} {
	switch fieldDataType {
	// Text data types
	case "CHAR":
		return &sql.NullString{}
	case "VARCHAR":
		return &sql.NullString{}
	case "TEXT":
		return &sql.NullString{}
	case "BYTEA":
		return &sql.NullString{}
	// Number data types
	case "INT2":
		return &sql.NullInt64{}
	case "INT4":
		return &sql.NullInt64{}
	case "INT8":
		return &sql.NullInt64{}
	case "NUMERIC":
		return &sql.NullFloat64{}
	case "MONEY":
		return &sql.NullFloat64{}
	// Date data types
	case "TIMESTAMP":
		return &sqlBackend.NullTime{}
	case "TIMESTAMPTZ":
		return &sqlBackend.NullTime{}
	case "DATE":
		return &sqlBackend.NullTime{}
	case "TIME":
		return &sqlBackend.NullTime{}
	case "TIMETZ":
		return &sqlBackend.NullTime{}
	// Other data types
	case "BOOL":
		return &sql.NullBool{}
	default:
		return &sql.NullString{}
	}
}
