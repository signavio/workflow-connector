package backend

import (
	"database/sql"
	"database/sql/driver"
	"net/http"
	"net/http/httptest"
	"net/url"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
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
type TestCase struct {
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
	// A testCase contains the expected relationship results that are associated
	// with this table
	ExpectedResultsRelationships []interface{}
	// A testCase contains the test data that a client would submit in an
	// HTTP POST
	PostData url.Values
	// A testCase contains a *http.Request
	Request func() *http.Request
	// run the testcase
	Run func(tc TestCase, ts *httptest.Server) error
}
