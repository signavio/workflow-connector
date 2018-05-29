package mysql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/backend/sql"
)

var (
	queryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			"   = _{{$.TableName}}.{{.UniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = ?",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}}" +
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
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.{{.UniqueIDColumn}}{{end}} LIMIT 1",
	}
)

func NewMysqlBackend() (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend()
	b.ConvertDBSpecificDataType = convertFromMysqlDataType
	b.Templates = queryTemplates
	return b
}

func convertFromMysqlDataType(fieldDataType string) interface{} {
	switch fieldDataType {
	case "number":
		return &sql.NullInt64{}
	case "text":
		return &sql.NullString{}
	case "real":
		return &sql.NullFloat64{}
	case "date":
		return &sqlBackend.NullTime{}
	default:
		return &sql.NullString{}
	}
}
