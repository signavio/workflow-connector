package sql

import (
	"database/sql/driver"
	"net/http"
	"net/url"
	"strings"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// These test cases don't really test if the mock database is updated
// since that really isnt supported by go-sqlmock. We can however
// test that the updated resource is returned upon a successfull
// update
var testCasesUpdateSingle = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when provided with valid parameters as URL parameters",
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
		RowsAsCsv: "2,Sanremo Café Racer,9283.99,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 9283.99,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00Z",
  "recipes": [%s]
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "equipment": "2",
      "id": "1",
      "instructions": "do this",
      "name": "Espresso single shot"
    }
  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("Sanremo Café Racer", "9283.99", "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("name", "Sanremo Café Racer")
			postData.Set("acquisitionCost", "9283.99")
			req, _ := http.NewRequest("PATCH", "/equipment/2?"+postData.Encode(), nil)
			return req
		},
	},
	{
		Kind: "success",
		Name: "it succeeds when provided with valid parameters as json in the request body",
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
		RowsAsCsv: "2,Sanremo Café Racer,9283.99,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 9283.99,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00Z",
  "recipes": [%s]
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "equipment": "2",
      "id": "1",
      "instructions": "do this",
      "name": "Espresso single shot"
    }
	  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("Sanremo Café Racer", 9283.99, "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			body := strings.NewReader(
				`{"name": "Sanremo Café Racer", "acquisitionCost": 9283.99}`,
			)
			req, _ := http.NewRequest(
				"PATCH",
				"/equipment/2",
				body,
			)
			req.Header.Set("Content-Type", "application/json")
			return req
		},
	},
	{

		Kind: "failure",
		Name: "it fails and returns 404 NOT FOUND when trying to update a non existent id",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonMaintenanceDescriptorFields,
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
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (200L)", "512.23", "42").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("name", "HolzbierFaß (200L)")
			postData.Set("acquisitionCost", "512.23")
			req, _ := http.NewRequest("PATCH", "/equipment/42?"+postData.Encode(), nil)
			return req
		},
	},
}
