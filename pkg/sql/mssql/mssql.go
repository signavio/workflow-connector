package mssql

import (
	"database/sql"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/signavio/workflow-connector/pkg/config"
	sqlBackend "github.com/signavio/workflow-connector/pkg/sql"
)

func NewMssqlBackend(cfg *config.Config) (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend(cfg)
	b.ConvertDBSpecificDataType = convertFromMssqlDataType
	b.Queries = map[string]string{
		"Get":                              "SELECT * FROM %s WHERE id = @p1",
		"GetCollection":                    "SELECT * FROM %s",
		"GetSingleAsOption":                "SELECT id, %s FROM %s WHERE id = @p1",
		"GetCollectionAsOptions":           "SELECT id, %s FROM %s",
		"GetCollectionAsOptionsFilterable": "SELECT id, %s FROM %s WHERE %s LIKE @p1",
		"GetTableSchema":                   "SELECT TOP 1 * FROM %s",
	}
	b.Templates = map[string]string{
		"UpdateSingle": "UPDATE {{.Table}} SET {{.ColumnNames | head}}" +
			" = @p1{{range $index, $element := .ColumnNames | tail}}," +
			" {{$element}} = @p{{(add2 $index)}}{{end}}" +
			" WHERE id = @p{{(lenPlus1 .ColumnNames)}}",
		"CreateSingle": "INSERT INTO {{.Table}}({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}, {{.}}{{end}}) VALUES(@p1{{range $index," +
			" $element := .ColumnNames | tail}}, @p{{$index | add2}}{{end}})",
	}
	return b
}

func convertFromMssqlDataType(fieldDataType string) interface{} {
	switch fieldDataType {
	// Text data types
	case "CHAR":
		return &sql.NullString{}
	case "VARCHAR":
		return &sql.NullString{}
	case "TEXT":
		return &sql.NullString{}
	case "NCHAR":
		return &sql.NullString{}
	case "NVARCHAR":
		return &sql.NullString{}
	case "NTEXT":
		return &sql.NullString{}
	case "BINARY":
		return &sql.NullString{}
	case "VARBINARY":
		return &sql.NullString{}
	case "IMAGE":
		return &sql.NullString{}
	// Number data types
	case "TINYINT":
		return &sql.NullInt64{}
	case "SMALLINT":
		return &sql.NullInt64{}
	case "INT":
		return &sql.NullInt64{}
	case "BIGINT":
		return &sql.NullInt64{}
	case "DECIMAL":
		return &sql.NullFloat64{}
	case "NUMERIC":
		return &sql.NullFloat64{}
	case "SMALLMONEY":
		return &sql.NullFloat64{}
	case "MONEY":
		return &sql.NullFloat64{}
	case "FLOAT":
		return &sql.NullFloat64{}
	case "REAL":
		return &sql.NullFloat64{}
	// Date data types
	case "DATETIME":
		return &sqlBackend.NullTime{}
	case "DATETIME2":
		return &sqlBackend.NullTime{}
	case "DATETIMEOFFSET":
		return &sqlBackend.NullTime{}
	case "SMALLDATETIME":
		return &sqlBackend.NullTime{}
	case "DATE":
		return &sqlBackend.NullTime{}
	case "TIME":
		return &sqlBackend.NullTime{}
	default:
		return &sql.NullString{}
	}
}
