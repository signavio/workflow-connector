package mysql

import (
	"database/sql"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/sql"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

const (
	dateTimeMysqlFormat = `'%Y-%m-%dT%TZ'`
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
		"GetCollectionAsOptionsWithParams": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE {{.ColumnAsOptionName}} LIKE ? " +
			"{{range $key, $value := .ParamsWithValues}}" +
			"AND {{$key}} = '{{$value}}'" +
			"{{end}}",
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

type Mysql struct {
	*sqlBackend.SqlBackend
}

func New() endpoint.Endpoint {
	m := &Mysql{sqlBackend.New().(*sqlBackend.SqlBackend)}
	m.Templates = QueryTemplates
	m.CastBackendTypeToGolangType = convertFromMysqlDataType
	return m
}

func convertFromMysqlDataType(fieldDataType string) interface{} {
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
