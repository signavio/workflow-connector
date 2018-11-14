package mysql

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	QueryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			"   = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = ?",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE {{.UniqueIDColumn}} = ?",
		"GetCollection": "SELECT * " +
			"FROM {{.TableName}}",
		"GetCollectionFilterable": "SELECT * " +
			"FROM {{.TableName}} " +
			"WHERE {{.FilterOnColumn}} {{.Operator}} ?",
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
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			" = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}{{end}} LIMIT 1",
	}
	integer = []string{
		"BIGINT",
		"INT",
		"INTEGER",
		"MEDIUMINT",
		"SMALLINT",
		"TINYINT",
	}
	text = []string{
		"BLOB",
		"TEXT",
		"VARCHAR",
		"CHAR",
		"TINYBLOB",
		"TINYTEXT",
		"MEDIUMBLOB",
		"MEDIUMTEXT",
		"LARGEBLOB",
		"LARGETEXT",
		"ENUM",
	}
	numeric = []string{
		"DECIMAL",
		"DOUBLE",
		"FLOAT",
	}
	dateTime = []string{
		"DATE",
		"DATETIME",
		"TIME",
		"TIMESTAMP",
		"YEAR",
	}
)

func ExecContextDirectly(ctx context.Context, db *sql.DB, query string, args ...interface{}) (result sql.Result, err error) {
	result, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return
}
func ExecContextWithinTx(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (result sql.Result, err error) {
	result, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return
}

func ConvertFromMysqlDataType(fieldDataType string) interface{} {
	switch {
	case isOfDataType(integer, fieldDataType):
		return &sql.NullInt64{}
	case isOfDataType(text, fieldDataType):
		return &sql.NullString{}
	case isOfDataType(numeric, fieldDataType):
		return &sql.NullFloat64{}
	case isOfDataType(dateTime, fieldDataType):
		return &util.NullTime{}
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

func InjectFormattingFuncs(query string, columnNames []string, fields []*config.Field) (queryWithFormatting string) {
	return query
}
