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
			commonMaintenanceDescriptorFields,
		},
		TableSchema: commonEquipmentTableSchema,
		ColumnNames: []string{
			"equipment\x00id",
			"equipment\x00name",
		},
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L)\n" +
			"2,HolzbierFaß (200L)\n" +
			"3,Refractometer",
		ExpectedResults: `[
  {
    "id": "1",
    "name": "Stainless Steel Mash Tun (50L)"
  },
  {
    "id": "2",
    "name": "HolzbierFaß (200L)"
  },
  {
    "id": "3",
    "name": "Refractometer"
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
