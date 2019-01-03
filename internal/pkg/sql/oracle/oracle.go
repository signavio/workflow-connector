package oracle

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/sql"
	"github.com/signavio/workflow-connector/internal/pkg/util"
	"gopkg.in/goracle.v2"
)

type lastId struct {
	id int64
}

const (
	dateTimeOracleFormat = `'YYYY-MM-DD"T"HH24:MI:SSXFF3TZH:TZM'`
	dateTimeGolangFormat = `2006-01-02T15:04:05.999-07:00`
)

var (
	QueryTemplates = map[string]string{
		`GetSingle`: `SELECT * ` +
			`FROM {{.TableName}} "_{{.TableName}}"` +
			`{{range .Relations}}` +
			`   LEFT JOIN {{.Relationship.WithTable}}` +
			`   ON {{.Relationship.WithTable}}."{{.Relationship.ForeignTableUniqueIDColumn}}"` +
			`   = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIDColumn}}"` +
			`{{end}}` +
			` WHERE "_{{$.TableName}}"."{{.UniqueIDColumn}}" = :1`,
		`GetSingleAsOption`: `SELECT "{{.UniqueIDColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM {{.TableName}} ` +
			`WHERE "{{.UniqueIDColumn}}" = :1`,
		`GetCollection`: `SELECT * ` +
			`FROM {{.TableName}}`,
		`GetCollectionFilterable`: `SELECT * ` +
			`FROM {{.TableName}} ` +
			`WHERE "{{.FilterOnColumn}}" {{.Operator}} :1`,
		`GetCollectionAsOptions`: `SELECT "{{.UniqueIDColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM {{.TableName}}`,
		`GetCollectionAsOptionsFilterable`: `SELECT "{{.UniqueIDColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM {{.TableName}} ` +
			`WHERE UPPER("{{.ColumnAsOptionName}}") LIKE '%'||UPPER(:1)||'%'`,
		`UpdateSingle`: `UPDATE {{.TableName}} ` +
			`SET "{{.ColumnNames | head}}" = :1` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  "{{$element}}" = :{{(add2 $index)}}` +
			`{{end}} ` +
			`WHERE "{{.UniqueIDColumn}}"= :{{(lenPlus1 .ColumnNames)}}`,
		`CreateSingle`: `DECLARE "l_{{.UniqueIDColumn}}" nvarchar2(256); ` +
			`BEGIN ` +
			`INSERT INTO {{.TableName}}` +
			`("{{.ColumnNames | head}}"` +
			`{{range .ColumnNames | tail}},` +
			`  "{{.}}"` +
			`{{end}}) ` +
			`VALUES(:1` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  :{{$index | add2}}` +
			`{{end}}) RETURNING "{{.UniqueIDColumn}}" INTO "l_{{.UniqueIDColumn}}"; ` +
			`DBMS_OUTPUT.PUT_LINE("l_{{.UniqueIDColumn}}"); ` +
			`END;`,
		`DeleteSingle`: `DELETE FROM {{.TableName}} WHERE "{{.UniqueIDColumn}}" = :1`,
		`GetTableSchema`: `SELECT * ` +
			`FROM {{.TableName}} ` +
			`WHERE ROWNUM <= 1`,
		`GetTableWithRelationshipsSchema`: `SELECT * FROM {{.TableName}} "_{{.TableName}}"` +
			`{{range .Relations}}` +
			` LEFT JOIN {{.Relationship.WithTable}}` +
			` ON {{.Relationship.WithTable}}."{{.Relationship.ForeignTableUniqueIDColumn}}"` +
			` = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIDColumn}}"{{end}} WHERE ROWNUM <= 1`,
	}
	integer = []string{}
	text    = []string{
		"CHAR",
		"NCHAR",
		"VARCHAR",
		"VARCHAR2",
		"NVARCHAR2",
		"LONG",
		"CLOB",
		"NCLOB",
	}
	numeric = []string{
		"NUMBER",
		"BINARY_DOUBLE",
		"BINARY_FLOAT",
		"FLOAT",
	}
	dateTime = []string{
		"TIMESTAMP",
		"TIMESTAMP WITH TIME ZONE",
		"TIMESTAMP WITH LOCAL TIME ZONE",
		"DATE",
	}
	boolean = []string{
		"BOOL",
	}
)

type Oracle struct {
	*sqlBackend.SqlBackend
}

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return 0, nil
}
func New() endpoint.Endpoint {
	o := &Oracle{sqlBackend.New().(*sqlBackend.SqlBackend)}
	o.Templates = QueryTemplates
	o.ExecContextFunc = wrapExecContext(o.DB, o.ExecContextFunc)
	o.CastDatabaseTypeToGolangType = convertFromOracleDataType
	o.CoerceExecArgsFunc = coerceExecArgsToOracleType
	o.OpenFunc = o.Open
	return o
}
func driverSpecificInitialization(ctx context.Context, db *sql.DB) error {
	log.When(config.Options.Logging).Infoln("[db] Performing driver specific initialization")
	if err := goracle.EnableDbmsOutput(ctx, db); err != nil {
		return err
	}
	return nil
}
func (o *Oracle) Open(args ...interface{}) error {
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		config.Options.Database.Driver,
	)
	driver := args[0].(string)
	url := args[1].(string)
	log.When(config.Options.Logging).Infof(
		"[backend] open connection to database %v\n",
		driver,
	)
	db, err := sql.Open(driver, url)
	if err != nil {
		return fmt.Errorf("Error opening connection to database: %s", err)
	}
	o.DB = db
	if err := driverSpecificInitialization(context.Background(), o.DB); err != nil {
		log.When(config.Options.Logging).Infof("Error performing driver specific initialization: %s", err)
		return fmt.Errorf("Error performing driver specific initialization: %s", err)
	}
	err = o.SaveSchemaMapping()
	if err != nil {
		return fmt.Errorf("Error saving table schema: %s", err)
	}
	return nil
}
func wrapExecContext(db *sql.DB, execContext func(context.Context, string, ...interface{}) (sql.Result, error)) func(context.Context, string, ...interface{}) (sql.Result, error) {
	return func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
		lastInserted := bytes.NewBufferString("")
		var id int64
		var formattedArgs []interface{}
		for _, arg := range args {
			formattedArgs = append(formattedArgs, formatArg(arg))
		}
		log.When(config.Options.Logging).Infof(
			"[handler -> db] The following query: \n%s\nwill be executed with these args:\n%s\n",
			query,
			formattedArgs,
		)
		result, err := execContext(ctx, query, args...)
		if err := goracle.ReadDbmsOutput(ctx, lastInserted, db); err != nil {
			return nil, err
		}
		if lastInserted.String() != "" {
			id, err = strconv.ParseInt(chomp(lastInserted.String()), 10, 64)
			if err != nil {
				return nil, err
			}
		}
		result = &lastId{id}
		return result, nil
	}
}
func convertFromOracleDataType(fieldDataType string) interface{} {
	switch {
	case isOfDataType(integer, fieldDataType):
		return &sql.NullInt64{}
	case isOfDataType(text, fieldDataType):
		return &sql.NullString{}
	case isOfDataType(numeric, fieldDataType):
		return &sql.NullFloat64{}
	case isOfDataType(dateTime, fieldDataType):
		return &util.NullTime{}
	case isOfDataType(boolean, fieldDataType):
		return &sql.NullBool{}
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

func formatArg(arg interface{}) (formattedArg interface{}) {
	switch v := arg.(type) {
	case time.Time:
		return v.Format(dateTimeGolangFormat)
	default:
		return v
	}
}

func coerceExecArgsToOracleType(query string, columnNames []string, fields []*descriptor.Field) (queryWithFormatting string) {
	queryWithFormatting = query
	for _, field := range fields {
		for i, column := range columnNames {
			if field.FromColumn == column || field.Type.Amount.FromColumn == column {
				queryParamToWrap := fmt.Sprintf(":%v", i+1)
				re := regexp.MustCompile(queryParamToWrap)
				switch field.Type.Kind {
				case "datetime", "date", "time":
					queryWithFormatting = re.ReplaceAllString(
						query, fmt.Sprintf("to_timestamp_tz(%s, %s)", queryParamToWrap, dateTimeOracleFormat),
					)
				}
			}
		}
	}
	return
}

func chomp(s string) string {
	return s[0:strings.IndexRune(s, '\n')]
}
