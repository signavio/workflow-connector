package sql

import (
	"database/sql/driver"
	"net/http"
	"net/url"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesCreateSingle = []testCase{
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
		RowsAsCsv: "4,Cooling Spiral,99.99,2017-03-02T00:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 99.99,
    "currency": "EUR"
  },
  "id": "4",
  "name": "Cooling Spiral",
  "purchaseDate": "2017-03-02T00:00:00Z",
  "recipes": [%s]
}`,
		ExpectedResultsRelationships: []interface{}{""},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO (.+)\\(id, name, acquisition_cost, purchase_date\\) VALUES\\(., ., ., .\\)").
				// insert id specifically instead of relying on the autoincrement feature
				// of the database. This allows us to run our tests multiple times on
				// the test database in such a way that the state of the database
				// before running the tests *is equal to* the state after
				// runnning the tests
				WithArgs("4", "Cooling Spiral", "99.99", "2017-03-02T00:00:00Z").
				WillReturnResult(sqlmock.NewResult(4, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("4").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("id", "4")
			postData.Set("name", "Cooling Spiral")
			postData.Set("acquisitionCost", "99.99")
			postData.Set("purchaseDate", "2017-03-02T00:00:00Z")
			req, _ := http.NewRequest("POST", "/equipment?"+postData.Encode(), nil)
			return req
		},
	},
}
