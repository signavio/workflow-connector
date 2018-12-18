package postgres

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/lib/pq"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	sqlBackend "github.com/signavio/workflow-connector/internal/pkg/sql"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type lastId struct {
	id int64
}

var (
	QueryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			"   LEFT JOIN {{.Relationship.WithTable}}" +
			"   ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			"   = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}" +
			"{{end}}" +
			" WHERE _{{$.TableName}}.{{.UniqueIDColumn}} = $1",
		"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE {{.UniqueIDColumn}} = $1",
		"GetCollection": "SELECT * " +
			"FROM {{.TableName}} " +
			"ORDER BY {{.UniqueIDColumn}} ASC",
		"GetCollectionFilterable": "SELECT * " +
			"FROM {{.TableName}} " +
			"WHERE {{.FilterOnColumn}} {{.Operator}} $1",
		"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"ORDER BY {{.UniqueIDColumn}} ASC",
		"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
			"FROM {{.TableName}} " +
			"WHERE CAST ({{.ColumnAsOptionName}} AS TEXT) ILIKE $1",
		"UpdateSingle": "UPDATE {{.TableName}} " +
			"SET {{.ColumnNames | head}} = $1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  {{$element}} = ${{(add2 $index)}}" +
			"{{end}} " +
			"WHERE {{.UniqueIDColumn}}= ${{(lenPlus1 .ColumnNames)}}",
		"CreateSingle": "INSERT INTO {{.TableName}}" +
			"({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}," +
			"  {{.}}" +
			"{{end}}) " +
			"VALUES($1" +
			"{{range $index, $element := .ColumnNames | tail}}," +
			"  ${{$index | add2}}" +
			"{{end}}) " +
			"RETURNING {{.UniqueIDColumn}}",
		"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = $1",
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
		"INT2",
		"INT4",
		"INT8",
	}
	text = []string{
		"CHAR",
		"VARCHAR",
		"TEXT",
		"BYTEA",
	}
	numeric = []string{
		"NUMERIC",
		"MONEY",
	}
	dateTime = []string{
		"TIMESTAMP",
		"TIMESTAMPTZ",
		"DATE",
		"TIME",
		"TIMETZ",
	}
	boolean = []string{
		"BOOL",
	}
)

type Postgres struct {
	*sqlBackend.SqlBackend
}

func (l *lastId) LastInsertId() (int64, error) {
	return l.id, nil
}

func (l *lastId) RowsAffected() (int64, error) {
	return 0, nil
}

func New() endpoint.Endpoint {
	p := &Postgres{sqlBackend.New().(*sqlBackend.SqlBackend)}
	p.ExecContextFunc = execContext(p.SqlBackend)
	p.Templates = QueryTemplates
	p.CastDatabaseTypeToGolangType = convertFromPostgresDataType
	return p
}

func execContext(b *sqlBackend.SqlBackend) func(context.Context, string, ...interface{}) (sql.Result, error) {
	return func(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
		var id int64
		requestTx := ctx.Value(util.ContextKey("tx")).(string)
		if requestTx == "" {
			// User has not specified an existing transaction to execute within.
			// However, we will still run the exec statement within a new
			// transaction
			tx, err := b.DB.Begin()
			if err != nil {
				return nil, err
			}
			defer func() {
				if p := recover(); p != nil {
					tx.Rollback()
					panic(p) // re-throw panic after tx.Rollback()
				} else if err != nil {
					tx.Rollback()
				} else {
					err = tx.Commit()
				}
			}()
		} else {
			// We assume the transacation is a valid one
			txi, _ := b.Transactions.Load(requestTx)
			tx := txi.(*sql.Tx)
			if err = b.DB.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
				return nil, err
			}
			result = &lastId{id}
			tx.Commit()
			return
		}
		if err = b.DB.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			return nil, err
		}
		result = &lastId{id}
		return
	}
}
func convertFromPostgresDataType(fieldDataType string) interface{} {
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
