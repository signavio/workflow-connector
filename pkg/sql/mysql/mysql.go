package mysql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/pkg/config"
	sqlBackend "github.com/signavio/workflow-connector/pkg/sql"
)

func NewMysqlBackend(cfg *config.Config, router *mux.Router) (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend(cfg, router)
	b.ConvertDBSpecificDataType = convertFromMysqlDataType
	b.Queries = map[string]string{
		"Get":                              "SELECT * FROM %s WHERE id = ?",
		"GetCollection":                    "SELECT * FROM %s",
		"GetSingleAsOption":                "SELECT id, %s FROM %s WHERE id = ?",
		"GetCollectionAsOptions":           "SELECT id, %s FROM %s",
		"GetCollectionAsOptionsFilterable": "SELECT id, %s FROM %s WHERE %s LIKE ?",
		"GetTableSchema":                   "SELECT * FROM %s LIMIT 1",
	}
	b.Templates = map[string]string{
		"UpdateSingle": "UPDATE {{.Table}} SET {{.ColumnNames | head}}" +
			" = ?{{range .ColumnNames | tail}}, {{.}} = ?{{end}} WHERE id = ?",
		"CreateSingle": "INSERT INTO {{.Table}}({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}, {{.}}{{end}}) " +
			"VALUES(?{{range .ColumnNames | tail}}, ?{{end}})",
	}
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
