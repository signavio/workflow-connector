package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesGetCollectionAsOptions = []testCase{
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
		RowsAsCsv: "1,Bialetti Moka Express 6 cup\n" +
			"2,Sanremo Café Racer\n" +
			"3,Buntfink SteelKettle\n" +
			"4,Copper Coffee Pot Cezve",
		ExpectedResults: `[
  {
    "id": "1",
    "name": "Bialetti Moka Express 6 cup"
  },
  {
    "id": "2",
    "name": "Sanremo Café Racer"
  },
  {
    "id": "3",
    "name": "Buntfink SteelKettle"
  },
  {
    "id": "4",
    "name": "Copper Coffee Pot Cezve"
  }
]`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT (.+), (.+) FROM (.+)").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/options", nil)
			return req
		},
	},
}
