package sqlite

import (
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	QueryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}} " +
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
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}} " +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			" = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}{{end}} LIMIT 1",
	}
	integer = []string{
		"BIGINT",
		"INT",
		"INT2",
		"INT8",
		"INTEGER",
		"MEDIUMINT",
		"SMALLINT",
		"TINYINT",
		"UNSIGNED",
	}
	text = []string{
		"CHARACTER",
		"CLOB",
		"NATIVE CHARACTER",
		"NCHAR",
		"NVARCHAR",
		"TEXT",
		"VARCHAR",
		"VARYING CHARACTER",
	}
	real = []string{
		"DOUBLE PRECISION",
		"DOUBLE",
		"FLOAT",
		"REAL",
	}
	numeric = []string{
		"BOOLEAN",
		"DECIMAL",
		"NUMERIC",
	}
	dateTime = []string{
		"DATE",
		"DATETIME",
	}
)

func ConvertFromSqliteDataType(fieldDataType string) interface{} {
	switch {
	case isOfDataType(integer, fieldDataType):
		return &sql.NullInt64{}
	case isOfDataType(text, fieldDataType):
		return &sql.NullString{}
	case isOfDataType(real, fieldDataType):
		return &sql.NullFloat64{}
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
