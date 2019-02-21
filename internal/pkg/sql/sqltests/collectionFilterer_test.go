package sqltests

import (
	"net/http"
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
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 39.95,
    "currency": "EUR"
  },
  "id": "3",
  "name": "Buntfink SteelKettle",
  "purchaseDate": "2017-12-12T12:00:00.000Z"
}`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment?filter=name+eq+Buntfink+SteelKettle", nil)
				return req
			},
		},
	}
)
