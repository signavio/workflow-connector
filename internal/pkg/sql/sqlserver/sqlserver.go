package sqlserver

import (
	"database/sql"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/sql"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	QueryTemplates = map[string]string{
		`GetSingle`: `SELECT * ` +
			`FROM "{{.TableName}}" AS "_{{.TableName}}" ` +
			`{{range .Relations}}` +
			`   LEFT JOIN "{{.Relationship.WithTable}}"` +
			`   ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			`   = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}}"` +
			`{{end}}` +
			`WHERE "_{{$.TableName}}"."{{.UniqueIdColumn}}" = @p1`,
		`GetSingleAsOption`: `SELECT "{{.UniqueIdColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM "{{.TableName}}" ` +
			`WHERE "{{.UniqueIdColumn}}" = @p1`,
		`GetCollection`: `SELECT * ` +
			`FROM "{{.TableName}}" AS "_{{.TableName}}" ` +
			`{{range .Relations}}` +
			`   LEFT JOIN "{{.Relationship.WithTable}}"` +
			`   ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			`   = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}}"` +
			`{{end}}` +
			"{{with .ColumnNames}}" +
			"   WHERE `_{{$.TableName}}`.`{{. | head}}` = @p1 " +
			"   {{range $index, $element := . | tail}}" +
			"      AND `_{{$.TableName}}`.`{{$element}}` = @p{{(add2 $index)}} " +
			"   {{end}}" +
			"{{end}}",
		`GetCollectionAsOptions`: `SELECT "{{.UniqueIdColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM "{{.TableName}}" ` +
			`WHERE CAST ("{{.ColumnAsOptionName}}" AS TEXT) LIKE @p1 ` +
			`{{range $index, $element := .ColumnNames}}` +
			`   AND "_{{$.TableName}}"."{{$element}}" = @p{{(add2 $index)}}` +
			`{{end}}`,
		`UpdateSingle`: `UPDATE "{{.TableName}}" ` +
			`SET "{{.ColumnNames | head}}" = @p1` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  "{{$element}}" = @p{{(add2 $index)}}` +
			`{{end}} ` +
			`WHERE "{{.UniqueIdColumn}}"= @p{{(lenPlus1 .ColumnNames)}}`,
		`CreateSingle`: `INSERT INTO "{{.TableName}}"` +
			`("{{.ColumnNames | head}}"` +
			`{{range .ColumnNames | tail}},` +
			`  "{{.}}"` +
			`{{end}}) ` +
			`VALUES(@p1` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  @p{{$index | add2}}` +
			`{{end}}) ` +
			`RETURNING "{{.UniqueIdColumn}}"`,
		`DeleteSingle`: `DELETE FROM "{{.TableName}}" WHERE "{{.UniqueIdColumn}}" = ?`,
		`GetTableSchema`: `SELECT TOP 1 * ` +
			`FROM "{{.TableName}}"`,
		`GetTableWithRelationshipsSchema`: `SELECT TOP 1 * FROM "{{.TableName}}" AS "_{{.TableName}}"` +
			`{{range .Relations}}` +
			` LEFT JOIN "{{.Relationship.WithTable}}"` +
			` ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			` = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}}"{{end}}`,
	}
	integer = []string{
		"TINYINT",
		"SMALLINT",
		"INT",
		"BIGINT",
	}
	text = []string{
		"CHAR",
		"VARCHAR",
		"TEXT",
		"NCHAR",
		"NVARCHAR",
		"NTEXT",
		"BINARY",
		"VARBINARY",
		"IMAGE",
	}
	numeric = []string{
		"DECIMAL",
		"NUMERIC",
		"SMALLMONEY",
		"MONEY",
		"FLOAT",
		"REAL",
	}
	dateTime = []string{
		"DATETIME",
		"DATETIME2",
		"DATETIMEOFFSET",
		"SMALLDATETIME",
		"DATE",
		"TIME",
	}
)

type Sqlserver struct {
	*sqlBackend.SqlBackend
}

func New() endpoint.Endpoint {
	s := &Sqlserver{sqlBackend.New().(*sqlBackend.SqlBackend)}
	s.Templates = QueryTemplates
	return s
}

func ConvertFromSqlserverDataType(fieldDataType string) interface{} {
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
