package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesGetCollectionAsOptionsFilterable = []testCase{
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
		},
		RowsAsCsv: "1,Bialetti Moka Express 6 cup",
		ExpectedResults: `[
  {
    "id": "1",
    "name": "Bialetti Moka Express 6 cup"
  }
]`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT (.+), (.+) FROM (.+) WHERE (.+) LIKE (.+)").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/options?filter=moka", nil)
			return req
		},
	},
}
