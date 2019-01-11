package sqltests

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var (
	collectionFiltererTests = map[string][]testCase{
		"GetCollectionFilterable": getCollectionFilterableTestCases,
	}
	getCollectionFilterableTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when filtering equipment table using column name",
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
			RowsAsCsv: "3,Buntfink SteelKettle,39.95,2017-12-12T12:00:00Z",
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 39.95,
    "currency": "EUR"
  },
  "id": "3",
  "name": "Buntfink SteelKettle",
  "purchaseDate": "2017-12-12T12:00:00.000Z"
}`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) WHERE name = .").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment?filter=name+eq+Buntfink+SteelKettle", nil)
				return req
			},
		},
	}
)
