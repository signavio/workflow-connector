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
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L),999,2017-12-12T12:00:00Z\n" +
			"2,HolzbierFaß (200L),512.23,2017-12-12T12:00:00Z\n" +
			"3,Refractometer,129,2017-12-12T12:00:00Z",
		ExpectedResults: `[
  {
    "acquisitionCost": {
      "amount": 999,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Stainless Steel Mash Tun (50L)",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 512.23,
      "currency": "EUR"
    },
    "id": "2",
    "name": "HolzbierFaß (200L)",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 129,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Refractometer",
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
