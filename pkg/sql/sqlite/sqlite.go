package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sdaros/workflow-db-connector/pkg/config"
	sqlBackend "github.com/sdaros/workflow-db-connector/pkg/sql"
)

func NewSqliteBackend(cfg *config.Config) (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend(cfg)
	b.ConvertDBSpecificDataType = convertFromSqliteDataType
	b.Queries = map[string]string{
		"GetSingleAsOption":                "SELECT id, %s FROM %s WHERE id = ?",
		"GetCollection":                    "SELECT * FROM %s",
		"GetCollectionAsOptions":           "SELECT id, %s FROM %s",
		"GetCollectionAsOptionsFilterable": "SELECT id, %s FROM %s WHERE %s LIKE ?",
		"GetTableSchema":                   "SELECT * FROM %s LIMIT 1",
	}
	b.Templates = map[string]string{
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.id{{end}} LIMIT 1",
		"GetSingleWithRelationships": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.id{{end}}" +
			" WHERE _{{$.TableName}}.id = ?",
		"UpdateSingle": "UPDATE {{.Table}} SET {{.ColumnNames | head}}" +
			" = ?{{range .ColumnNames | tail}}, {{.}} = ?{{end}} WHERE id = ?",
		"CreateSingle": "INSERT INTO {{.Table}}({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}, {{.}}{{end}}) " +
			"VALUES(?{{range .ColumnNames | tail}}, ?{{end}})",
	}
	return b
}

func convertFromSqliteDataType(fieldDataType string) interface{} {
	switch fieldDataType {
	case "number":
		return &sql.NullInt64{}
	case "integer":
		return &sql.NullInt64{}
	case "text":
		return &sql.NullString{}
	case "real":
		return &sql.NullFloat64{}
	case "date":
		return &sqlBackend.NullTime{}
	case "datetime":
		return &sqlBackend.NullTime{}
	default:
		return &sql.NullString{}
	}
}
