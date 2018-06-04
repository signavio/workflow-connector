package sql

import (
	"database/sql/driver"
	"net/http"
	"net/url"
	"strings"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesUpdateSingle = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when provided with valid parameters as URL parameters",
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
		RowsAsCsv: "2,HolzbierFaß (100L),299.99,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 299.99,
    "currency": "EUR"
  },
  "id": "2",
  "maintenance": [%s],
  "name": "HolzbierFaß (100L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "comments": "It went poorly!",
      "datePerformed": "2018-02-03T12:22:01Z",
      "dateScheduled": "2017-02-03T02:00:00Z",
      "equipmentId": 2,
      "id": "2",
      "maintainerId": 1
    }
  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (100L)", "299.99", "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("name", "HolzbierFaß (100L)")
			postData.Set("acquisitionCost", "299.99")
			req, _ := http.NewRequest("PATCH", "/equipment/2?"+postData.Encode(), nil)
			return req
		},
	},
	{
		Kind: "success",
		Name: "it succeeds when provided with valid parameters as json in the request body",
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
		RowsAsCsv: "2,HolzbierFaß (200L),512.23,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 512.23,
    "currency": "EUR"
  },
  "id": "2",
  "maintenance": [%s],
  "name": "HolzbierFaß (200L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "comments": "It went poorly!",
      "datePerformed": "2018-02-03T12:22:01Z",
      "dateScheduled": "2017-02-03T02:00:00Z",
      "equipmentId": 2,
      "id": "2",
      "maintainerId": 1
    }
  `},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (200L)", 512.23, "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			body := strings.NewReader(
				`{"name": "HolzbierFaß (200L)", "acquisitionCost": 512.23}`,
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
