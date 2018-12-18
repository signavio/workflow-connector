package sqltests

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/signavio/workflow-connector/internal/pkg/sql/mysql"
	"github.com/signavio/workflow-connector/internal/pkg/sql/oracle"
	"github.com/signavio/workflow-connector/internal/pkg/sql/postgres"
	"github.com/signavio/workflow-connector/internal/pkg/sql/sqlite"
	"github.com/signavio/workflow-connector/internal/pkg/sql/sqlserver"
	"github.com/spf13/viper"
)

var (
	commonEquipmentTableSchema = &descriptor.SchemaMapping{
		FieldNames: []string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		GolangTypes: []interface{}{
			&sql.NullString{String: "", Valid: true},
			&sql.NullString{String: "", Valid: true},
			&sql.NullFloat64{Float64: 0, Valid: true},
			&sql.NullString{String: "", Valid: true},
		},
	}
	commonRecipesTableSchema = &descriptor.SchemaMapping{
		FieldNames: []string{
			"recipes\x00id",
			"recipes\x00equipment_id",
			"recipes\x00name",
			"recipes\x00instructions",
		},
		GolangTypes: []interface{}{
			&sql.NullString{String: "", Valid: true},
			&sql.NullFloat64{Float64: 0, Valid: true},
			&sql.NullString{String: "", Valid: true},
			&sql.NullString{String: "", Valid: true},
		},
	}

	descriptorFileBase = `
{
  "key": "test",
  "name": "Test",
  "description": "Just a test",
  "typeDescriptors": [
    {
      "key" : "equipment",
      "name" : "Equipment",
      "tableName": "equipment",
      "columnAsOptionName": "name",
      "uniqueIdColumn": "id",
      "fields" : [
        %s
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    },
    {
      "key" : "recipes",
      "name" : "Recipes",
      "tableName": "recipes",
      "columnAsOptionName": "name",
      "uniqueIdColumn": "id",
      "fields" : [
        %s
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ]
}
`
	commonEquipmentDescriptorFields = `
{
  "key" : "id",
  "name" : "ID",
  "fromColumn": "id",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "name",
  "name" : "Equipment Name",
  "fromColumn": "name",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "acquisitionCost",
  "name" : "Acquisition Cost",
  "type" : {
	"name" : "money",
	"amount" : {
      "key": "acquisitionCost",
	  "fromColumn": "acquisition_cost"
	},
	"currency" : {
	  "value" : "EUR"
	}
  }
},
{
  "key" : "purchaseDate",
  "name" : "Purchase Date",
  "fromColumn" : "purchase_date",
  "type" : {
	"name" : "date",
	"kind" : "date"
  }
},
{
  "key" : "recipes",
  "name" : "Associated recipes",
  "type" : {
  	"name": "text"
  },
  "relationship": {
  	"kind": "oneToMany",
  	"withTable": "recipes",
  	"localTableUniqueIdColumn": "id",
  	"foreignTableUniqueIdColumn": "equipment_id"
  }
}`
	commonRecipesDescriptorFields = `
{
  "key" : "id",
  "name" : "Recipe ID",
  "fromColumn": "id",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "name",
  "name" : "Recipe name",
  "fromColumn": "name",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "instructions",
  "name" : "Instructions",
  "fromColumn": "instructions",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "equipmentId",
  "name" : "Equipment ID",
  "fromColumn": "equipment_id",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "equipment",
  "name" : "Equipment",
  "type" : {
    "name": "text"
  },
  "relationship": {
    "kind": "manyToOne",
    "withTable": "equipment",
    "localTableUniqueIdColumn": "equipment_id",
    "foreignTableUniqueIdColumn": "id"
  }
}`
	queryTemplates = map[string]string{
		"GetSingle": "SELECT * " +
			"  FROM {{.TableName}} AS _{{.TableName}} " +
			"  {{range .Relations}}" +
			"     LEFT JOIN {{.Relationship.WithTable}}" +
			"     ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			"     = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}" +
			"  {{end}}" +
			"  WHERE _{{$.TableName}}.{{$.UniqueIDColumn}} = ?",
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
		"GetTableWithRelationshipsSchema": "SELECT * " +
			"FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
			" = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}{{end}} LIMIT 1",
	}
)

// TestCase for sql backend
type testCase struct {
	// A testCase should assert success cases or failure cases
	Kind string
	// A testCase has a unique name
	Name string
	// A testCase has descriptor fields that describe the schema of the
	// mocked database table in workflow accelerator's custom json format
	DescriptorFields []string
	// A testCase has a tableSchema that describes the schema of the
	// mocked database table using golang's native data types
	TableSchema *descriptor.SchemaMapping
	// A testCase contains an array with the names of all columns in the
	// mocked database table
	ColumnNames []string
	// A testCase contains the row data for each column in the mocked database
	// table in csv format
	RowsAsCsv string
	// A testCase contains the SQL queries which should be executed against
	// the mock database
	ExpectedQueries func(sqlmock.Sqlmock, []string, string, ...driver.Value)
	// A testCase contains the expected results that should be returned after
	// the database has been queried and the results are processed
	// by the formatter
	ExpectedResults string
	// A testCase contains the expected http status code that should be
	// returned to the client
	ExpectedStatusCode int
	// A testCase contains the expected key-value pairs present in the http
	// header that is returned to the client
	ExpectedHeader http.Header
	// A testCase contains the expected relationship results that are associated
	// with this table
	ExpectedResultsRelationships []interface{}
	// A testCase contains the test data that a client would submit in an
	// HTTP POST
	PostData url.Values
	// A testCase contains a *http.Request
	Request func() *http.Request
	// run the testcase
	Run func(tc testCase, ts *httptest.Server) error
}

func TestSqlBackends(t *testing.T) {
	var testUsingDB string
	if viper.IsSet("db") {
		testUsingDB = viper.Get("db").(string)
	}
	/*	if strings.Contains(testUsingDB, "mock") {
		t.Run("Using mocked database", func(t *testing.T) {
			ts, backend, mock, err := testOnMockedDB()
			if err != nil {
				t.Errorf(err.Error())
			}
			defer ts.Close()
			for handlerName, testCases := range handlerTests {
				t.Run(handlerName, func(t *testing.T) {
					for _, tc := range testCases {
						// The config.Descriptor in config.Options needs to be mocked
						mockedDescriptorFile, err := mockDescriptorFile(tc.DescriptorFields...)
						if err != nil {
							t.Errorf("unexpected error: %v", err)
							return
						}
						config.Options.Descriptor = config.ParseDescriptorFile(mockedDescriptorFile)
						// mock the database table schema
						backend.TableSchemas = make(map[string]*TableSchema)
						backend.TableSchemas["equipment"] = tc.TableSchema
						backend.TableSchemas["equipment\x00relationships"] = tc.TableSchema
						backend.TableSchemas["recipes"] = tc.TableSchema
						backend.TableSchemas["recipes\x00relationships"] = tc.TableSchema
						backend.TransactDirectly = func(ctx context.Context, db *sql.DB, query string, args ...interface{}) (result sql.Result, err error) {
							result, err = db.ExecContext(ctx, query, args...)
							if err != nil {
								return nil, err
							}
							return
						}
						// initialize mock database
						tc.ExpectedQueries(mock, tc.ColumnNames, tc.RowsAsCsv)
						t.Run(tc.Name, func(t *testing.T) {
							tc.setExpectedResults(handlerName, true)
							err := run(tc, ts)
							if err != nil {
								t.Errorf(err.Error())
								return
							}
							if mockErr := mock.ExpectationsWereMet(); mockErr != nil {
								t.Errorf("there were unfulfilled expectations: %s", mockErr)
								return
							}

						})
					}
				})
			}
		})
	}*/
	switch {
	case strings.Contains(testUsingDB, "sqlite"):
		testSqlBackend(t, "sqlite", "sqlite3", sqlite.New)
	case strings.Contains(testUsingDB, "mysql"):
		testSqlBackend(t, "mysql", "mysql", mysql.New)
	case strings.Contains(testUsingDB, "oracle"):
		testSqlBackend(t, "oracle", "goracle", oracle.New)
	case strings.Contains(testUsingDB, "sqlserver"):
		testSqlBackend(t, "sqlserver", "sqlserver", sqlserver.New)
	case strings.Contains(testUsingDB, "postgres"):
		testSqlBackend(t, "postgres", "postgres", postgres.New)
	default:
		testSqlBackend(t, "sqlite", "sqlite3", sqlite.New)
	}
}

func testSqlBackend(t *testing.T, name, driver string, newEndpointFunc func() endpoint.Endpoint) {
	endpoint := newEndpointFunc()
	err := endpoint.Open(
		driver,
		viper.Get(name+".database.url").(string),
	)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Run("Using "+name+" database", func(t *testing.T) {
		ts := newTestServer(endpoint)
		defer ts.Close()
		for testName, testCases := range conformityTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range crudTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range dataConnectorOptionsTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range collectionFiltererTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}

	})
}
func runTestCases(t *testing.T, testName string, testCases []testCase, ts *httptest.Server, endpoint endpoint.Endpoint) {
	t.Run(testName, func(t *testing.T) {
		for _, tc := range testCases {
			ts := newTestServer(endpoint)
			defer ts.Close()
			t.Run(tc.Name, func(t *testing.T) {
				tc.setExpectedResults(testName, false)
				err := run(tc, ts)
				if err != nil {
					t.Errorf(err.Error())
					return
				}
			})
		}
	})
}

func (tc *testCase) setExpectedResults(testName string, isMockDB bool) {
	switch testName {
	case "GetSingle", "UpdateSingle", "CreateSingle":
		if isMockDB {
			var emptyRelationships []interface{}
			for i := 0; i < len(tc.ExpectedResultsRelationships); i++ {
				emptyRelationships = append(emptyRelationships, "")
			}

			tc.ExpectedResults = fmt.Sprintf(
				tc.ExpectedResults,
				emptyRelationships...,
			// TODO maybe we have to populate the array
			// "",
			)

			return
		}
		tc.ExpectedResults = fmt.Sprintf(
			tc.ExpectedResults,
			tc.ExpectedResultsRelationships...,
		)
	}
}
func run(tc testCase, ts *httptest.Server) error {
	switch tc.Kind {
	case "success":
		tc.Run = itSucceeds
		if err := tc.Run(tc, ts); err != nil {
			return err
		}
		return nil
	case "failure":
		tc.Run = itFails
		if err := tc.Run(tc, ts); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("testcase should either be success or failure kind")
	}
}
func itFails(tc testCase, ts *httptest.Server) error {
	req := tc.Request()
	u, err := url.Parse(ts.URL + req.URL.RequestURI())
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	req.URL = u
	req.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	if res.StatusCode == tc.ExpectedStatusCode {
		return fmt.Errorf(
			"expected code '%d' , instead we received: %d",
			tc.ExpectedStatusCode,
			res.StatusCode,
		)
	}
	if string(got[:]) != tc.ExpectedResults {
		return fmt.Errorf(
			"response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s",
			got,
			tc.ExpectedResults,
		)
	}
	return nil
}

func itSucceeds(tc testCase, ts *httptest.Server) error {
	req := tc.Request()
	u, err := url.Parse(ts.URL + req.URL.RequestURI())
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	req.URL = u
	req.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}

	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	if tc.ExpectedStatusCode != 0 {
		if res.StatusCode != tc.ExpectedStatusCode {
			return fmt.Errorf(
				"expected HTTP %d, instead we received: %d",
				tc.ExpectedStatusCode,
				res.StatusCode,
			)
		}
	}
	if tc.ExpectedHeader != nil {
		if res.Header.Get("Location") == tc.ExpectedHeader.Get("Location") {
			return fmt.Errorf(
				"expected HTTP Header %s, instead we received: %s",
				res.Header.Get("Location"),
				tc.ExpectedHeader.Get("Location"),
			)
		}
	}
	if string(got[:]) != tc.ExpectedResults {
		return fmt.Errorf(
			"response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s",
			got,
			tc.ExpectedResults,
		)
	}
	return nil
}

//func testOnMockedDB() (ts *httptest.Server, b *Backend, mock sqlmock.Sqlmock, err error) {
//	b = NewBackend("sqlmock")
//	b.Templates = queryTemplates
//	b.DB, mock, err = sqlmock.New()
//	if err != nil {
//		return nil, nil, mock, fmt.Errorf(
//			"an error '%s' was not expected when opening a stub database connection",
//			err,
//		)
//	}
//	ts = newTestServer(b)
//	return
//}
func newTestServer(e endpoint.Endpoint) *httptest.Server {
	router := e.GetHandler().(*mux.Router)
	ts := httptest.NewUnstartedServer(router)
	router.Use(middleware.BasicAuth)
	router.Use(middleware.RouteChecker)
	router.Use(middleware.RequestInjector)
	router.Use(middleware.ResponseInjector)
	server := &http.Server{}
	server.Handler = router
	ts.Config = server
	ts.Start()
	return ts
}
func mockDescriptorFile(testCaseDescriptorFields ...string) (io.Reader, error) {
	equipmentDescriptorFields := testCaseDescriptorFields[0]
	recipesDescriptorFields := testCaseDescriptorFields[1]
	mockedDescriptorFile := fmt.Sprintf(
		descriptorFileBase,
		equipmentDescriptorFields,
		recipesDescriptorFields,
	)
	return strings.NewReader(mockedDescriptorFile), nil
}
