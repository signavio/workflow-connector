package oracle

import (
	"context"
	"database/sql"
	"fmt"
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

const (
	dateTimeOracleFormat   = `'YYYY-MM-DD"T"HH24:MI:SSXFF3TZH:TZM'`
	dateTimeGolangFormat   = `2006-01-02T15:04:05.000-07:00`
	dateOracleFormat       = `'YYYY-MM-DD'`
	dateGolangFormat       = `2006-01-02`
	timeOracleFormat       = `'YYYY-MM-DD"T"HH24:MI:SSXFF3'`
	timeGolangFormat       = `2006-01-02T15:04:05.000`
	dateTimeWorkflowFormat = `2006-01-02T15:04:05.000Z`
)

type Oracle struct {
	*sqlBackend.SqlBackend
	characterSet    characterSet
	sessionTimeZone *time.Location
}
type lastId struct {
	id           int64
	rowsAffected int64
}
type characterSet encoding.Encoding

var (
	Universal         characterSet = unicode.UTF8
	EuroSymbolSupport characterSet = charmap.Windows1252
	QueryTemplates                 = map[string]string{
		`GetSingle`: `SELECT * ` +
			`FROM "{{.TableName}}" "_{{.TableName}}"` +
			`{{range .Relations}}` +
			`   LEFT JOIN "{{.Relationship.WithTable}}"` +
			`   ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			`   = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}}"` +
			`{{end}}` +
			` WHERE "_{{$.TableName}}"."{{.UniqueIdColumn}}" = :1`,
		`GetSingleAsOption`: `SELECT "{{.UniqueIdColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM "{{.TableName}}" ` +
			`WHERE "{{.UniqueIdColumn}}" = :1`,
		`GetCollection`: `SELECT * ` +
			`FROM "{{.TableName}}" "_{{.TableName}}"` +
			`{{range .Relations}}` +
			`   LEFT JOIN "{{.Relationship.WithTable}}"` +
			`   ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			`   = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}} "` +
			`{{end}}` +
			`{{with .ColumnNames}}` +
			`   WHERE "_{{$.TableName}}"."{{. | head}} = :1 ` +
			`   {{range $index, $element := . | tail}}` +
			`      AND "_{{$.TableName}}"."{{$element}}" = {{(format $index $element)}} ` +
			`   {{end}}` +
			`{{end}}`,
		`GetCollectionAsOptions`: `SELECT "{{.UniqueIdColumn}}", "{{.ColumnAsOptionName}}" ` +
			`FROM "{{.TableName}}" ` +
			`WHERE UPPER("{{.ColumnAsOptionName}}") LIKE '%'||UPPER(:1)||'%' ` +
			`{{range $index, $element := .ColumnNames}}` +
			`   AND "_{{$.TableName}}"."{{$element}}" = {{(format $index $element)}} ` +
			`{{end}}`,
		`UpdateSingle`: `UPDATE "{{.TableName}}" ` +
			`{{with $firstColumn := .ColumnNames | head}}` +
			`SET "{{$firstColumn}}" = {{(format -1 $firstColumn)}}` +
			`{{end}}` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  "{{$element}}" = {{(format $index $element)}}` +
			`{{end}} ` +
			`WHERE "{{.UniqueIdColumn}}"= :{{(lenPlus1 .ColumnNames)}}`,
		`CreateSingle`: `DECLARE "l_{{.UniqueIdColumn}}" nvarchar2(256); ` +
			`BEGIN ` +
			`INSERT INTO "{{.TableName}}"` +
			`("{{.ColumnNames | head}}"` +
			`{{range .ColumnNames | tail}},` +
			`  "{{.}}"` +
			`{{end}}) ` +
			`{{with $firstColumn := .ColumnNames | head}}` +
			`VALUES({{(format -1 $firstColumn)}}` +
			`{{end}}` +
			`{{range $index, $element := .ColumnNames | tail}},` +
			`  {{format $index $element}}` +
			`{{end}}) RETURNING "{{.UniqueIdColumn}}" INTO "l_{{.UniqueIdColumn}}"; ` +
			`DBMS_OUTPUT.PUT_LINE("l_{{.UniqueIdColumn}}"); ` +
			"END;",
		`DeleteSingle`: `DELETE FROM "{{.TableName}}" WHERE "{{.UniqueIdColumn}}" = :1`,
		`GetTableSchema`: `SELECT * ` +
			`FROM "{{.TableName}}" ` +
			`WHERE ROWNUM <= 1`,
		`GetTableWithRelationshipsSchema`: `SELECT * FROM "{{.TableName}}" "_{{.TableName}}"` +
			`{{range .Relations}}` +
			` LEFT JOIN "{{.Relationship.WithTable}}"` +
			` ON "{{.Relationship.WithTable}}"."{{.Relationship.ForeignTableUniqueIdColumn}}"` +
			` = "_{{$.TableName}}"."{{.Relationship.LocalTableUniqueIdColumn}}"{{end}} WHERE ROWNUM <= 1`,
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
	oracleQueryFormatFuncs = map[string]func() string{
		"datetime": func() string {
			dateTimeOracleFormat := `'YYYY-MM-DD"T"HH24:MI:SSXFF3TZH:TZM'`
			return fmt.Sprintf(
				"to_timestamp_tz(%%s, %s)",
				dateTimeOracleFormat,
			)
		},
		"date": func() string {
			dateOracleFormat := `'YYYY-MM-DD'`
			return fmt.Sprintf(
				"to_date(%%s, %s)",
				dateOracleFormat,
			)
		},
		"time": func() string {
			timeOracleFormat := `'YYYY-MM-DD"T"HH24:MI:SSXFF3'`
			return fmt.Sprintf(
				"to_date(%%s, %s)",
				timeOracleFormat,
			)
		},
		"default": func() string {
			return "%s"
		},
	}
	oracleDateTimeArgFunc = func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool, err error) {
		dateTimeWorkflowFormat := `2006-01-02T15:04:05.000Z`
		dateTimeGolangFormat := `2006-01-02T15:04:05.000-07:00`
		if result, ok := requestData[field.Key]; ok {
			if result != nil {
				stringifiedDateTime := result.(string)
				parsedDateTime, err := time.ParseInLocation(
					dateTimeWorkflowFormat, stringifiedDateTime, time.UTC,
				)
				if err != nil {
					return nil, ok, err
				}
				formattedDateTime := parsedDateTime.Format(dateTimeGolangFormat)
				return formattedDateTime, ok, nil
			}
			return result, ok, nil
		}
		return
	}
	oracleDateArgFunc = func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool, err error) {
		dateTimeWorkflowFormat := `2006-01-02T15:04:05.000Z`
		dateGolangFormat := `2006-01-02`
		if result, ok := requestData[field.Key]; ok {
			if result != nil {
				stringifiedDateTime := result.(string)
				parsedDateTime, err := time.ParseInLocation(
					dateTimeWorkflowFormat, stringifiedDateTime, time.UTC,
				)
				if err != nil {
					return nil, ok, err
				}
				formattedDateTime := parsedDateTime.Format(dateGolangFormat)
				return formattedDateTime, ok, nil
			}
			return result, ok, nil
		}
		return
	}
	oracleTimeArgFunc = func(requestData map[string]interface{}, field *descriptor.Field) (result interface{}, ok bool, err error) {
		dateTimeWorkflowFormat := `2006-01-02T15:04:05.000Z`
		timeGolangFormat := `2006-01-02T15:04:05.000`
		if result, ok := requestData[field.Key]; ok {
			if result != nil {
				stringifiedDateTime := result.(string)
				parsedDateTime, err := time.ParseInLocation(
					dateTimeWorkflowFormat, stringifiedDateTime, time.UTC,
				)
				if err != nil {
					return nil, ok, err
				}
				formattedDateTime := parsedDateTime.Format(timeGolangFormat)
				return formattedDateTime, ok, nil
			}
			return result, ok, nil
		}
		return
	}
)

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return l.rowsAffected, nil
}
func New() endpoint.Endpoint {
	// Assume UTF-8 character set before checking
	o := &Oracle{sqlBackend.New().(*sqlBackend.SqlBackend), Universal, time.UTC}
	o.Templates = QueryTemplates
	o.CastBackendTypeToGolangType = convertFromOracleDataType
	oracleSpecificArgFuncs := o.CoerceArgFuncs
	oracleSpecificArgFuncs["datetime"] = oracleDateTimeArgFunc
	oracleSpecificArgFuncs["date"] = oracleDateArgFunc
	oracleSpecificArgFuncs["time"] = oracleTimeArgFunc
	o.CoerceArgFuncs = oracleSpecificArgFuncs
	o.QueryFormatFuncs = oracleQueryFormatFuncs
	o.NewSchemaMapping = o.newOracleSchemaMapping
	o.OpenFunc = o.Open
	return o
}
func (o *Oracle) Open(args ...interface{}) error {
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
	if err := driverSpecificInitialization(context.Background(), o); err != nil {
		return fmt.Errorf("Error performing driver specific initialization: %s", err)
	}
	o.QueryContextFunc = wrapQueryContext(o.characterSet, o.QueryContextFunc)
	o.ExecContextFunc = wrapExecContext(o, o.ExecContextFunc)
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
			if ok && scale < 1 {
				// The goracle driver in use treats a NUMBER(p,0) as a float64
				// even if the scale == 0, this makes stringifying an id of
				// type NUMBER(38,0) a pain since it appears as "1.00000"
				// when the id == 1 for example
				backendType = "INTEGER"
			}
		}
		golangType := o.CastBackendTypeToGolangType(backendType)
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
func (o *Oracle) setCharacterSet() (err error) {
	log.When(config.Options.Logging).Infoln("[oracle] query characterset in use by db")
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
func driverSpecificInitialization(ctx context.Context, o *Oracle) error {
	log.When(config.Options.Logging).Infoln("[oracle] performing driver specific initialization")
	if err := goracle.EnableDbmsOutput(ctx, o.DB); err != nil {
		return err
	}
	if err := o.setCharacterSet(); err != nil {
		return err
	}
	if err := o.setSessionTimeZone(); err != nil {
		return err
	}
	return nil
}
func (o *Oracle) setSessionTimeZone() error {
	getSessionTimeZone :=
		`SELECT SESSIONTIMEZONE FROM DUAL`
	var sessionTimeZone string
	err := o.DB.QueryRowContext(context.Background(), getSessionTimeZone).Scan(&sessionTimeZone)
	if err != nil {
		return err
	}
	log.When(config.Options.Logging).
		Infof("[oracle] current session time zone is: %s\n", sessionTimeZone)
	parsedTime, err := time.Parse("-07:00", sessionTimeZone)
	if err != nil {
		return err
	}
	o.sessionTimeZone = parsedTime.Location()
	return nil
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
func wrapExecContext(o *Oracle, execContext func(context.Context, string, ...interface{}) (sql.Result, error)) func(context.Context, string, ...interface{}) (sql.Result, error) {
	return func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
		//lastInserted := bytes.NewBufferString("")
		var id int64
		result, err := execContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		rowsAffected, _ := result.RowsAffected()
		//	if err := goracle.ReadDbmsOutput(ctx, lastInserted, db); err != nil {
		//		return nil, err
		//	}
		//	if lastInserted.String() != "" {
		//		id, err = strconv.ParseInt(chomp(lastInserted.String()), 10, 64)
		//		if err != nil {
		//			return nil, err
		//		}
		//	}
		result = &lastId{id, rowsAffected}
		return result, nil
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
func chomp(s string) string {
	return s[0:strings.IndexRune(s, '\n')]
}
