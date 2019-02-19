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
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"gopkg.in/goracle.v2"
)

type lastId struct {
	id int64
}
type characterSet encoding.Encoding

const (
	dateTimeOracleFormat = `'YYYY-MM-DD"T"HH24:MI:SSXFF3TZH:TZM'`
	dateTimeGolangFormat = `2006-01-02T15:04:05.999-07:00`
)

var (
	Universal         characterSet = unicode.UTF8
	EuroSymbolSupport characterSet = charmap.Windows1252
	QueryTemplates                 = map[string]string{
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
		`GetCollectionAsOptionsWithParams`: `SELECT "{{.UniqueIDColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM {{.TableName}} ` +
			`WHERE UPPER("{{.ColumnAsOptionName}}") LIKE '%'||UPPER(:1)||'%' ` +
			`{{range $key, $value := .ParamsWithValues}}` +
			`AND "{{$key}}" = '{{$value}}'` +
			`{{end}}`,
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
	integer = []string{
		"INTEGER",
	}
	text = []string{
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
	characterSet characterSet
}

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return 0, nil
}
func New() endpoint.Endpoint {
	// Assume UTF-8 character set before checking
	o := &Oracle{sqlBackend.New().(*sqlBackend.SqlBackend), Universal}
	o.Templates = QueryTemplates
	o.ExecContextFunc = wrapExecContext(o.DB, o.ExecContextFunc)
	o.CastDatabaseTypeToGolangType = convertFromOracleDataType
	o.CoerceExecArgsFunc = coerceExecArgsToOracleType
	o.NewSchemaMapping = o.newOracleSchemaMapping
	o.OpenFunc = o.Open
	return o
}
func driverSpecificInitialization(ctx context.Context, db *sql.DB) error {
	log.When(config.Options.Logging).Infoln("[oracle] Performing driver specific initialization")
	if err := goracle.EnableDbmsOutput(ctx, db); err != nil {
		return err
	}
	return nil
}
func (o *Oracle) setCharacterSet() (err error) {
	getCharacterSet :=
		`SELECT VALUE FROM NLS_DATABASE_PARAMETERS WHERE PARAMETER = 'NLS_CHARACTERSET'`
	var charSet string
	err = o.DB.QueryRowContext(context.Background(), getCharacterSet).Scan(&charSet)
	if err != nil {
		log.When(config.Options.Logging).Infof("Error retrieving current character encoding from db: %s", err)
		return fmt.Errorf("Error retrieving current character encoding from db: %s", err)
	}
	switch charSet {
	case "AL32UTF8":
		o.characterSet = Universal
	case "WE8MSWIN1252":
		o.characterSet = EuroSymbolSupport
	default:
		// Unsupported character set
		return fmt.Errorf("Character set '%s' is not supported", charSet)
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
	if err = o.setCharacterSet(); err != nil {
		return err
	}
	o.QueryContextFunc = wrapQueryContext(o.characterSet, o.QueryContextFunc)
	err = o.SaveSchemaMapping()
	if err != nil {
		return fmt.Errorf("Error saving table schema: %s", err)
	}
	return nil
}
func (o *Oracle) newOracleSchemaMapping(columnsWithTable []string, columnTypes []*sql.ColumnType) (*descriptor.SchemaMapping, error) {
	var backendTypes, golangTypes, workflowTypes []interface{}
	var fieldNames []string
	for i := range columnTypes {
		backendType := columnTypes[i].DatabaseTypeName()
		if backendType == "" {
			return nil, fmt.Errorf(
				"unable to get the type in use by the backend",
			)
		} else if backendType == "NUMBER" {

			_, scale, ok := columnTypes[i].DecimalSize()
			if ok && scale == 0 {
				// The goracle driver in use treats a NUMBER(p,0) as a float64
				// even if the scale == 0, this makes stringifying an id of
				// type NUMBER(38,0) a pain since it appears as "1.00000"
				// when the id == 1 for example
				backendType = "INTEGER"
			}
		}
		golangType := o.CastDatabaseTypeToGolangType(backendType)
		if golangType == nil {
			return nil, fmt.Errorf(
				"unable to get the native golang type",
			)
		}
		workflowType := sqlBackend.GetWorkflowType(columnsWithTable[i])
		if workflowType == nil {
			return nil, fmt.Errorf(
				"unable to get the workflow type specified in descriptor.json",
			)
		}
		workflowTypes = append(workflowTypes, workflowType)
		backendTypes = append(backendTypes, backendType)
		golangTypes = append(golangTypes, golangType)
		fieldNames = append(fieldNames, columnsWithTable[i])
	}
	return &descriptor.SchemaMapping{fieldNames, backendTypes, golangTypes, workflowTypes}, nil
}

func wrapQueryContext(charSet characterSet, queryContext func(context.Context, string, ...interface{}) ([]interface{}, error)) func(context.Context, string, ...interface{}) ([]interface{}, error) {
	return func(ctx context.Context, query string, args ...interface{}) ([]interface{}, error) {
		var resultsAsUtf8 []interface{}
		results, err := queryContext(ctx, query, args...)
		for _, result := range results {
			resultAsUtf8, err := convertCharacterSetToUtf8(charSet, result)
			if err != nil {
				return nil, err
			}
			resultsAsUtf8 = append(resultsAsUtf8, resultAsUtf8)
		}
		return resultsAsUtf8, err
	}
}
func convertCharacterSetToUtf8(charSet characterSet, queryResult interface{}) (interface{}, error) {
	var err error
	utf8ResultOuter := make(map[string]interface{})
	tableAndRelationships := queryResult.(map[string]interface{})
	for ki, _ := range tableAndRelationships {
		utf8ResultInner := make(map[string]interface{})
		for kj, vj := range tableAndRelationships[ki].(map[string]interface{}) {
			switch vj.(type) {
			case string:
				utf8ResultInner[kj], err = charSet.NewEncoder().String(vj.(string))
				if err != nil {
					return nil, err
				}
			default:
				utf8ResultInner[kj] = vj
			}
		}
		utf8ResultOuter[ki] = utf8ResultInner
	}
	return utf8ResultOuter, nil
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
