package mysql

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
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
	m.CastDatabaseTypeToGolangType = convertFromMysqlDataType
	m.CoerceExecArgsFunc = coerceExecArgsToMysqlType
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

func coerceExecArgsToMysqlType(query string, columnNames []string, fields []*descriptor.Field) (queryWithFormatting string) {
	// We need the [^'"] at the end of the regular expression
	// to make sure that we do not match on column names
	// which may contain a literal ? in the name
	queryParamAndSeperator := `(?m)(?P<param>\?)(?P<seperator>[,;])[^'"]`
	pattern := regexp.MustCompile(queryParamAndSeperator)
	submatches := pattern.FindAllSubmatchIndex([]byte(query), -1)
	betweenTheMatches := pattern.Split(query, -1)
	coerceDateTemplate := []byte(fmt.Sprintf("str_to_date($param, %s)$seperator ", dateTimeMysqlFormat))
	doNothingTemplate := []byte("$param$seperator ")
	result := []byte{}
	for _, field := range fields {
		for i, column := range columnNames {
			if field.FromColumn == column || field.Type.Amount.FromColumn == column {
				switch field.Type.Kind {
				case "datetime", "date", "time":
					result = append(result, []byte(betweenTheMatches[i])...)
					result = pattern.Expand(result, coerceDateTemplate, []byte(query), submatches[i])
				default:
					result = append(result, []byte(betweenTheMatches[i])...)
					result = pattern.Expand(result, doNothingTemplate, []byte(query), submatches[i])
				}
			}
		}
	}
	result = append(result, []byte(betweenTheMatches[len(betweenTheMatches)-1])...)
	queryWithFormatting = string(result)
	return
}
