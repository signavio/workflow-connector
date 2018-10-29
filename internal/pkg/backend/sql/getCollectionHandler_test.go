package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesGetCollection = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when equipment table contains more than one column",
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
		RowsAsCsv: "1,Bialetti Moka Express 6 cup,25.95,2017-12-12T12:00:00Z\n" +
			"2,Sanremo Café Racer,8477.95,2017-12-12T12:00:00Z\n" +
			"3,Buntfink SteelKettle,39.95,2017-12-12T12:00:00Z\n" +
			"4,Copper Coffee Pot Cezve,49.95,2017-12-12T12:00:00Z",
		ExpectedResults: `[
  {
    "acquisitionCost": {
      "amount": 25.95,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Bialetti Moka Express 6 cup",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 8477.95,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 39.95,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Buntfink SteelKettle",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 49.95,
      "currency": "EUR"
    },
    "id": "4",
    "name": "Copper Coffee Pot Cezve",
    "purchaseDate": "2017-12-12T12:00:00Z"
  }
]`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+)").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment", nil)
			return req
		},
	},
}
