package sql

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
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/spf13/viper"
)

var (
	commonEquipmentTableSchema = &TableSchema{
		[]string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		[]interface{}{
			&sql.NullString{String: "", Valid: true},
			&sql.NullString{String: "", Valid: true},
			&sql.NullFloat64{Float64: 0, Valid: true},
			&sql.NullString{String: "", Valid: true},
		},
	}
	commonRecipesTableSchema = &TableSchema{
		[]string{
			"recipes\x00id",
			"recipes\x00equipment_id",
			"recipes\x00name",
			"recipes\x00instructions",
		},
		[]interface{}{
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

// testCases for sql backend
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
	TableSchema *TableSchema
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
type handlerTests map[string][]testCase

func TestHandlers(t *testing.T) {
	handlerTests := handlerTests{
		"GetSingleHandler":         testCasesGetSingle,
		"GetSingleAsOptionHandler": testCasesGetSingleAsOption,
		"GetCollectionHandler":     testCasesGetCollection,
		//	"GetCollectionHandlerFilterable":          testCasesGetCollectionFilterable,
		"GetCollectionAsOptionsHandler":           testCasesGetCollectionAsOptions,
		"GetCollectionAsOptionsFilterableHandler": testCasesGetCollectionAsOptionsFilterable,
		"UpdateSingleHandler":                     testCasesUpdateSingle,
		"CreateSingleHandler":                     testCasesCreateSingle,
		"DeleteSingleHandler":                     testCasesDeleteSingle,
	}
	var testUsingDB string
	defaultConfigOptions := config.Options
	if viper.IsSet("db") {
		testUsingDB = viper.Get("db").(string)
	}
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
	if strings.Contains(testUsingDB, "sqlite") &&
		viper.IsSet("sqlite.database.url") {
		// The default config.Descriptor should be used for real databases
		config.Options = defaultConfigOptions
		backend := NewBackend("sqlite")
		err := backend.Open("sqlite3", viper.Get("sqlite.database.url").(string))
		if err != nil {
			t.Errorf(err.Error())
			return
		}

		t.Run("Using sqlite database", func(t *testing.T) {
			ts := newTestServer(backend)
			defer ts.Close()
			for handlerName, testCases := range handlerTests {
				t.Run(handlerName, func(t *testing.T) {
					for _, tc := range testCases {
						t.Run(tc.Name, func(t *testing.T) {
							tc.setExpectedResults(handlerName, false)
							err := run(tc, ts)
							if err != nil {
								t.Errorf(err.Error())
								return
							}
						})
					}
				})
			}
		})
	}
	if strings.Contains(testUsingDB, "mysql") &&
		viper.IsSet("mysql.database.url") {
		backend := NewBackend("mysql")
		// The default config.Descriptor should be used for real databases
		config.Options = defaultConfigOptions
		err := backend.Open("mysql", viper.Get("mysql.database.url").(string))
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		t.Run("Using mysql database", func(t *testing.T) {
			ts := newTestServer(backend)
			defer ts.Close()
			for handlerName, testCases := range handlerTests {
				t.Run(handlerName, func(t *testing.T) {
					for _, tc := range testCases {
						ts := newTestServer(backend)
						defer ts.Close()
						t.Run(tc.Name, func(t *testing.T) {
							tc.setExpectedResults(handlerName, false)
							err := run(tc, ts)
							if err != nil {
								t.Errorf(err.Error())
								return
							}
						})
					}
				})
			}
		})
	}
	if strings.Contains(testUsingDB, "postgres") &&
		viper.IsSet("postgres.database.url") {
		// The default config.Descriptor should be used for real databases
		config.Options = defaultConfigOptions
		backend := NewBackend("postgres")
		err := backend.Open("postgres", viper.Get("postgres.database.url").(string))
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		t.Run("Using postgres database", func(t *testing.T) {
			ts := newTestServer(backend)
			defer ts.Close()
			for handlerName, testCases := range handlerTests {
				t.Run(handlerName, func(t *testing.T) {
					for _, tc := range testCases {
						ts := newTestServer(backend)
						defer ts.Close()
						t.Run(tc.Name, func(t *testing.T) {
							tc.setExpectedResults(handlerName, false)
							err := run(tc, ts)
							if err != nil {
								t.Errorf(err.Error())
								return
							}
						})
					}
				})
			}
		})
	}
}

func (tc *testCase) setExpectedResults(handlerName string, isMockDB bool) {
	switch handlerName {
	case "GetSingleHandler", "UpdateSingleHandler", "CreateSingleHandler":
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
	if res.StatusCode != 404 {
		return fmt.Errorf(
			"expected 404 Not Found, instead we received: %v",
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
	if strings.HasPrefix(string(res.StatusCode), "2") {
		return fmt.Errorf(
			"expected HTTP 2xx, instead we received: %d",
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

func testOnMockedDB() (ts *httptest.Server, b *Backend, mock sqlmock.Sqlmock, err error) {
	b = NewBackend("sqlmock")
	b.Templates = queryTemplates
	b.DB, mock, err = sqlmock.New()
	if err != nil {
		return nil, nil, mock, fmt.Errorf(
			"an error '%s' was not expected when opening a stub database connection",
			err,
		)
	}
	ts = newTestServer(b)
	return
}
func newTestServer(b *Backend) *httptest.Server {
	router := b.GetHandler().(*mux.Router)
	ts := httptest.NewUnstartedServer(router)
	router.Use(middleware.BasicAuth)
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
