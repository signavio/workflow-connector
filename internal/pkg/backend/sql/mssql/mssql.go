package mssql

import (
	"database/sql"

	_ "github.com/denisenkom/go-mssqldb"
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
			"WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = @p1",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE id = @p1",
		"GetCollection": "SELECT *" +
			"FROM {{.TableName}}",
		"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}}",
		"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE CAST ({{.ColumnAsOptionName}} AS TEXT LIKE @p1",
		"UpdateSingle": "UPDATE {{.TableName}} " +
			"SET {{.ColumnNames | head}} = @p1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  {{$element}} = @p{{(add2 $index)}}" +
			"{{end}} " +
			"WHERE {{.UniqueIDColumn}}= @p{{(lenPlus1 .ColumnNames)}}",
		"CreateSingle": "INSERT INTO {{.TableName}}" +
			"({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}," +
			"  {{.}}" +
			"{{end}}) " +
			"VALUES(@p1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  @p{{$index | add2}}" +
			"{{end}}) " +
			"RETURNING {{.UniqueIDColumn}}",
		"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = ?",
		"GetTableSchema": "SELECT TOP 1 * " +
			"FROM {{.TableName}}",
		"GetTableWithRelationshipsSchema": "SELECT TOP 1 * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.{{.UniqueIDColumn}}{{end}}",
	}
)

func NewMssqlBackend() (b *sqlBackend.Backend) {
	b = sqlBackend.NewBackend()
	b.ConvertDBSpecificDataType = convertFromMssqlDataType
	b.Templates = queryTemplates
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
