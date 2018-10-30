package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesGetSingle = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when equipment table contains more than one column",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonRecipesDescriptorFields,
		},
		TableSchema: commonEquipmentTableSchema,
		ColumnNames: []string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		RowsAsCsv: "2,Sanremo Café Racer,8477.85,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00Z",
  "recipes": [%s]
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "name": "Espresso single shot"
    }
  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/2", nil)
			return req
		},
	},
	{

		Kind: "failure",
		Name: "it fails and returns 404 NOT FOUND when querying a non existent equipment id",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonRecipesDescriptorFields,
		},
		TableSchema: commonEquipmentTableSchema,
		ColumnNames: []string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		RowsAsCsv:                    "",
		ExpectedResults:              `%s`,
		ExpectedResultsRelationships: []interface{}{""},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/42", nil)
			return req
		},
	},
	{
		Kind: "success",
		Name: "it succeeds when recipes table contains more than one column",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonRecipesDescriptorFields,
		},
		TableSchema: commonRecipesTableSchema,
		ColumnNames: []string{
			"recipes\x00id",
			"recipes\x00equipment_id",
			"recipes\x00name",
			"recipes\x00instructions",
		},
		RowsAsCsv: "1,2,Espresso single shot,do this",
		ExpectedResults: `{
  "equipment": {%s},
  "equipmentId": 2,
  "id": "1",
  "instructions": "do this",
  "name": "Espresso single shot"
}`,
		ExpectedResultsRelationships: []interface{}{`
    "acquisitionCost": {
      "amount": 8477.85,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00Z"
  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("1").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/recipes/1", nil)
			return req
		},
	},
}
