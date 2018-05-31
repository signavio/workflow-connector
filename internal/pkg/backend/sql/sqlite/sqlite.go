package sqlite

import (
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/backend/sql"
)

var (
	queryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}} " +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			"   = _{{$.TableName}}.{{.UniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = ?",
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
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}} " +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.{{.UniqueIDColumn}}{{end}} LIMIT 1",
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
	time = []string{
		"DATE",
		"DATETIME",
	}
)

func NewSqliteBackend() (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend()
	b.ConvertDBSpecificDataType = convertFromSqliteDataType
	b.Templates = queryTemplates
	return b
}

func convertFromSqliteDataType(fieldDataType string) interface{} {
	switch {
	case isOfDataType(integer, fieldDataType):
		return &sql.NullInt64{}
	case isOfDataType(text, fieldDataType):
		return &sql.NullString{}
	case isOfDataType(real, fieldDataType):
		return &sql.NullFloat64{}
	case isOfDataType(numeric, fieldDataType):
		return &sql.NullFloat64{}
	case isOfDataType(time, fieldDataType):
		return &sqlBackend.NullTime{}
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
