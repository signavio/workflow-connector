package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesDeleteSingle = []testCase{
	{
		Kind: "success",
		Name: "it succeeds in deleting an existing resource",
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
		RowsAsCsv: "",
		ExpectedResults: `{
  "status": {
    "code": 200,
    "description": "Resource with uniqueID '4' successfully deleted from equipment table"
  }
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM (.+) WHERE (.+) = (.+)").
				WithArgs("4").
				WillReturnResult(sqlmock.NewResult(4, 1))
			mock.ExpectCommit()
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("DELETE", "/equipment/4", nil)
			return req
		},
	},
	{

		Kind: "failure",
		Name: "it fails and returns 404 NOT FOUND when trying to delete a non existent id",
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
		RowsAsCsv: "",
		ExpectedResults: `{
  "errors": [
    {
      "code": 404,
      "description": "Resource with uniqueID '42' not found in equipment table"
    }
  ]
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("DELETE", "/equipment/42", nil)
			return req
		},
	},
}
