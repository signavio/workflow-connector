package sql

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	_ "io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"text/template"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/util"
)

const descriptorFileTemplateText = `
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
      "fields" : [
        {{.}}
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ]
}
`
const commonDescriptorFields = `
{
  "key" : "id",
  "name" : "ID",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "name",
  "name" : "Name",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "cost",
  "name" : "Cost",
  "type" : {
	"name" : "money",
	"amount" : {
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
}`

type equipment struct {
	id              *sql.NullString
	name            *sql.NullString
	acquisitionCost *sql.NullFloat64
	purchaseDate    *sql.NullString
}
type testCase struct {
	name             string
	descriptorFields string
	tableSchema      *config.TableSchema
	columnNames      []string
	rowsAsCsv        string
	expectations     func(sqlmock.Sqlmock, []string, string, ...driver.Value)
	expectedResults  []interface{}
	postData         url.Values
	request          *http.Request
}

var commonTableSchema = &config.TableSchema{
	[]string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
	[]interface{}{equip.id, equip.name, equip.acquisitionCost, equip.purchaseDate},
}
var equip = &equipment{
	id:              &sql.NullString{String: "", Valid: true},
	name:            &sql.NullString{String: "", Valid: true},
	acquisitionCost: &sql.NullFloat64{Float64: 0, Valid: true},
	purchaseDate:    &sql.NullString{String: "", Valid: true},
}

var testCasesForGetEquipmentByID = func(caseType string) []testCase {
	expectationsSuccess := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
			WithArgs("1").
			WillReturnRows(rows)
	}
	expectationsFail := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
			WithArgs("2").
			WillReturnRows(rows)
	}
	successCases := []testCase{
		{
			name:             "table with four columns",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99,2017-03-02T00:00:00Z",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.99,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
			},
			expectations: expectationsSuccess,
		},
		{
			name: "table with one column",
			descriptorFields: `
		{
		  "key" : "id",
		  "name" : "ID",
		  "type" : {
			"name" : "text"
		  }
		}`,
			tableSchema: &config.TableSchema{
				[]string{"equipment_id"},
				[]interface{}{equip.id},
			},
			columnNames: []string{"equipment_id"},
			rowsAsCsv:   "1",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id": "1",
					},
				},
			},
			expectations: expectationsSuccess,
		},
	}
	failureCases := []testCase{
		{
			name:             "table with four columns and incorrect row data",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
			rowsAsCsv:        "1,Wooden Keg (100L),400.99,2017-03-02T00:00:00Z",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.98,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
			},
			expectations: expectationsFail,
		},
	}
	switch caseType {
	case "success":
		return successCases
	case "failure":
		return failureCases
	}
	return []testCase{}
}

var testCasesForGetEquipmentOptionsByID = func(caseType string) []testCase {
	expectationsSuccess := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT id, name FROM (.+) WHERE (.+) = (.+)").
			WithArgs("1").
			WillReturnRows(rows)
	}
	expectationsFail := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
			WillReturnRows(rows)
	}
	successCases := []testCase{
		{
			name: "table with four columns",
			descriptorFields: `
	{
	  "key" : "id",
	  "name" : "ID",
	  "type" : {
	    "name" : "text"
	  }
	},
	{
	  "key" : "name",
	  "name" : "Name",
	  "type" : {
	    "name" : "text"
	  }
	},
	{
	  "key" : "cost",
	  "name" : "Cost",
	  "type" : {
	    "name" : "money",
	    "amount" : {
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
      "fromColumn": "purchase_date",
	  "type" : {
	    "name" : "date",
	    "kind" : "date"
	  }
	}`,
			tableSchema: &config.TableSchema{
				[]string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
				[]interface{}{equip.id, equip.name, equip.acquisitionCost, equip.purchaseDate},
			},
			columnNames: []string{"equipment_id", "equipment_name"},
			rowsAsCsv:   "1,HolzbierFaß (100L)",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":   "1",
						"name": "HolzbierFaß (100L)",
					},
				},
			},
			expectations: expectationsSuccess,
		},
	}
	failureCases := []testCase{
		{
			name:             "table with four columns and incorrect row data",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisitionCost"},
			rowsAsCsv:        "1,Wooden Keg (100L),400.99",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.98,
					},
				},
			},
			expectations: expectationsFail,
		},
	}
	switch caseType {
	case "success":
		return successCases
	case "failure":
		return failureCases
	}
	return []testCase{}
}

var testCasesForGetEquipmentCollectionOptions = func(caseType string) []testCase {
	expectationsSuccess := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT id, name FROM (.+)").
			WillReturnRows(rows)
	}
	expectationsFail := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
			WithArgs("2").
			WillReturnRows(rows)
	}
	successCases := []testCase{
		{
			name:             "table with four columns",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name"},
			rowsAsCsv:        "1,HolzbierFaß (100L)\n2,Cooling Spiral (2m)",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":   "1",
						"name": "HolzbierFaß (100L)",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":   "2",
						"name": "Cooling Spiral (2m)",
					},
				},
			},
			expectations: expectationsSuccess,
		},
	}
	failureCases := []testCase{
		{
			name:             "table with four columns and incorrect row data",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisition_cost"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99\n2,Cooling Spiral (2m),89.99",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":   "1",
						"name": "HolzbierFaß (100L)",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":   "1",
						"name": "HolzbierFaß (100L)",
					},
				},
			},
			expectations: expectationsFail,
		},
	}
	switch caseType {
	case "success":
		return successCases
	case "failure":
		return failureCases
	}
	return []testCase{}
}
var testCasesForGetEquipmentCollection = func(caseType string) []testCase {
	expectationsSuccess := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+)").
			WillReturnRows(rows)
	}
	expectationsFail := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
			WithArgs("2").
			WillReturnRows(rows)
	}
	successCases := []testCase{
		{name: "table with four columns",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99,2017-03-02T00:00:00Z\n2,Cooling Spiral (2m),89.99,2017-03-02T00:00:00Z",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.99,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "2",
						"name":             "Cooling Spiral (2m)",
						"acquisition_cost": 89.99,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
			},
			expectations: expectationsSuccess,
		},
		{
			name: "table with one column",
			descriptorFields: `
		{
		  "key" : "id",
		  "name" : "ID",
		  "type" : {
			"name" : "text"
		  }
		}`,
			tableSchema: &config.TableSchema{
				[]string{"equipment_id"},
				[]interface{}{equip.id},
			},
			columnNames: []string{"equipment_id"},
			rowsAsCsv:   "1\n2",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id": "1",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id": "2",
					},
				},
			},
			expectations: expectationsSuccess,
		},
	}
	failureCases := []testCase{
		{
			name:             "test with four columns and incorrect row data",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"id", "name", "acquisition_cost", "purchase_date"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99\n2,Cooling Spiral (2m),89.99",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.99,
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "2",
						"name":             "Cooling Spiral (2m)",
						"acquisition_cost": 89.99,
					},
				},
			},
			expectations: expectationsFail,
		},
	}
	switch caseType {
	case "success":
		return successCases
	case "failure":
		return failureCases
	}
	return []testCase{}
}
var testCasesForUpdateSingle = func(caseType string) []testCase {
	// NEXT: testCases for update single
	expectationsSuccess := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		rows := sqlmock.NewRows(columns).
			FromCSVString(rowsAsCsv)
		mock.ExpectExec("INSERT INTO (.+)").
			WithArgs(args)
		mock.ExpectQuery("SELECT . FROM (.+)").
			WillReturnRows(rows)
	}
	expectationsFail := func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
		mock.ExpectExec("INSERT INTO (.+)").
			WithArgs(args)
	}
	successCases := []testCase{
		{
			name:             "table with four columns",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"equipment_id", "equipment_name", "equipment_acquisition_cost", "equipment_purchase_date"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99,2017-03-02T00:00:00Z\n2,Cooling Spiral (2m),89.99,2017-03-02T00:00:00Z",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.99,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "2",
						"name":             "Cooling Spiral (2m)",
						"acquisition_cost": 89.99,
						"purchase_date":    "2017-03-02T00:00:00Z",
					},
				},
			},
			expectations: expectationsSuccess,
		},
		{
			name: "table with one column",
			descriptorFields: `
		{
		  "key" : "id",
		  "name" : "ID",
		  "type" : {
			"name" : "text"
		  }
		}`,
			tableSchema: &config.TableSchema{
				[]string{"equipment_id"},
				[]interface{}{equip.id},
			},
			columnNames: []string{"equipment_id"},
			rowsAsCsv:   "1\n2",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id": "1",
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id": "2",
					},
				},
			},
			expectations: expectationsSuccess,
		},
	}
	failureCases := []testCase{
		{
			name:             "test with four columns and incorrect row data",
			descriptorFields: commonDescriptorFields,
			tableSchema:      commonTableSchema,
			columnNames:      []string{"id", "name", "acquisition_cost", "purchase_date"},
			rowsAsCsv:        "1,HolzbierFaß (100L),400.99\n2,Cooling Spiral (2m),89.99",
			expectedResults: []interface{}{
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "1",
						"name":             "HolzbierFaß (100L)",
						"acquisition_cost": 400.99,
					},
				},
				map[string]interface{}{
					"equipment": map[string]interface{}{
						"id":               "2",
						"name":             "Cooling Spiral (2m)",
						"acquisition_cost": 89.99,
					},
				},
			},
			expectations: expectationsFail,
		},
	}
	switch caseType {
	case "success":
		return successCases
	case "failure":
		return failureCases
	}
	return []testCase{}
}

func setupBackend(t testCase) (b *Backend, mock sqlmock.Sqlmock, err error) {
	descriptorTemplate, err := template.New("test").Parse(descriptorFileTemplateText)
	if err != nil {
		return nil, nil, err
	}
	descriptorFileString := bytes.NewBufferString("")
	err = descriptorTemplate.Execute(descriptorFileString, t.descriptorFields)
	if err != nil {
		return nil, nil, err
	}
	cfg := config.Initialize(
		strings.NewReader(descriptorFileString.String()))
	b, mock, err = newTestBackend(cfg)
	if err != nil {
		return nil, nil, err
	}
	b.Cfg.TableSchemas["equipment"] = t.tableSchema
	return b, mock, nil

}

func newTestBackend(cfg *config.Config) (b *Backend, mock sqlmock.Sqlmock, err error) {
	b = NewBackend(cfg)
	b.Queries = map[string]string{
		"GetSingleAsOption":                "SELECT id, %s FROM %s WHERE id = ?",
		"GetCollection":                    "SELECT * FROM %s",
		"GetCollectionAsOptions":           "SELECT id, %s FROM %s",
		"GetCollectionAsOptionsFilterable": "SELECT id, %s FROM %s WHERE %s LIKE ?",
		"GetTableSchema":                   "SELECT * FROM %s LIMIT 1",
	}
	b.Templates = map[string]string{
		"GetTableWithRelationshipsSchema": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.id{{end}} LIMIT 1",
		"GetSingleWithRelationships": "SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.id{{end}}" +
			" WHERE _{{$.TableName}}.id = ?",
		"UpdateSingle": "UPDATE {{.Table}} SET {{.ColumnNames | head}}" +
			" = ?{{range .ColumnNames | tail}}, {{.}} = ?{{end}} WHERE id = ?",
		"CreateSingle": "INSERT INTO {{.Table}}({{.ColumnNames | head}}" +
			"{{range .ColumnNames | tail}}, {{.}}{{end}}) " +
			"VALUES(?{{range .ColumnNames | tail}}, ?{{end}})",
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, mock, fmt.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	b.DB = db
	return b, mock, nil
}

func handleTestCase(ctx context.Context, b *Backend, route string, t *testing.T) (response []interface{}, err error) {
	switch route {
	case "GetSingle":
		route := &getSingle{
			ctx:     ctx,
			id:      "1",
			backend: b,
		}
		return route.handle()
	case "GetCollection":
		route := &getCollection{
			ctx:     ctx,
			backend: b,
		}
		return route.handle()
	case "GetCollectionAsOptions":
		route := &getCollectionAsOptions{
			ctx:     ctx,
			backend: b,
			query: fmt.Sprintf(b.Queries["GetCollectionAsOptions"],
				ctx.Value(config.ContextKey("columnAsOptionName")).(string),
				"equipment"),
		}
		return route.handle()
	case "GetSingleAsOption":
		route := &getSingleAsOption{
			ctx:     ctx,
			id:      "1",
			backend: b,
			query: fmt.Sprintf(b.Queries["GetSingleAsOption"],
				ctx.Value(config.ContextKey("columnAsOptionName")).(string),
				"equipment"),
		}
		return route.handle()
	}
	return
}

func commonSubTest(tc testCase, t *testing.T) (*Backend, sqlmock.Sqlmock, context.Context) {
	b, mock, err := setupBackend(tc)
	if err != nil {
		t.Errorf("Expected no error, instand we received: %s", err)
	}
	ctx := util.BuildContext(
		context.Background(),
		b.Cfg.Descriptor.TypeDescriptors,
		"equipment")
	// Which SQL queries do we expect to be run?
	tc.expectations(mock, tc.columnNames, tc.rowsAsCsv)
	return b, mock, ctx
}

func commonTest(testCases func(string) []testCase, testName, route string, t *testing.T) {
	t.Run(testName, func(t *testing.T) {
		for _, tc := range testCases("success") {
			b, mock, ctx := commonSubTest(tc, t)
			t.Run(fmt.Sprintf("when using %v", tc.name), func(t *testing.T) {
				response, err := handleTestCase(ctx, b, route, t)
				if err != nil {
					t.Errorf("Expected no error, instead we received: %s", err)
				}
				if !reflect.DeepEqual(response, tc.expectedResults) {
					t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
						response, tc.expectedResults)
				}
				//Make sure that all expectations were met by the mock database
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expections: %s", err)
				}
			})
		}
	})
	t.Run("fails gracefully", func(t *testing.T) {
		for _, tc := range testCasesForGetEquipmentByID("failure") {
			b, mock, ctx := commonSubTest(tc, t)
			t.Run(fmt.Sprintf("when using %v", tc.name), func(t *testing.T) {
				response, err := handleTestCase(ctx, b, route, t)
				if err == nil {
					t.Errorf("Expected no error, instead we received: %s", err)
				}
				if reflect.DeepEqual(response, tc.expectedResults) {
					t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
						response, tc.expectedResults)
				}
				//Make sure some expectations were not met by the mock database
				if err := mock.ExpectationsWereMet(); err == nil {
					t.Errorf("There should be unfulfilled expectations left over: %s", err)
				}
			})
		}
	})
}
func TestGetSingleEquipment(t *testing.T) {
	commonTest(
		testCasesForGetEquipmentByID,
		"returns a single Equipment",
		"GetSingle",
		t,
	)
}
func TestGetSingleEquipmentAsOption(t *testing.T) {
	commonTest(
		testCasesForGetEquipmentOptionsByID,
		"returns a single equipment with property name and ID",
		"GetSingleAsOption",
		t,
	)
}
func TestGetEquipmentCollection(t *testing.T) {
	commonTest(
		testCasesForGetEquipmentCollection,
		"returns equipment collection",
		"GetCollection",
		t,
	)
}
func TestGetEquipmentCollectionAsOptions(t *testing.T) {
	commonTest(
		testCasesForGetEquipmentCollectionOptions,
		"returns equipment collection with property name and ID",
		"GetCollectionAsOptions",
		t,
	)
}

func TestGetEquipmentCollectionAsOptionsFilterable(t *testing.T) {
	//	commonTest(
	//		testCasesForGetEquipmentCollectionOptions,
	//		"returns equipment collection with property name and ID",
	//		"GetCollectionAsOptions",
	//		t,
	//	)
}
