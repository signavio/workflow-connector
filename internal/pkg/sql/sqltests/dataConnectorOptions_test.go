package sqltests

import (
	"net/http"
)

var (
	dataConnectorOptionsTests = map[string][]testCase{
		"GetSingleAsOption":                getSingleAsOptionTestCases,
		"GetCollectionAsOptions":           getCollectionAsOptionsTestCases,
		"GetCollectionAsOptionsFilterable": getCollectionAsOptionsFilterableTestCases,
		"GetCollectionAsOptionsWithParams": getCollectionAsOptionsWithParamsTestCases,
	}
	getSingleAsOptionTestCases = []testCase{
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
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "id": "1",
  "name": "Bialetti Moka Express 6 cup"
}`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options/1", nil)
				return req
			},
		},
		{
			Kind: "failure",
			Name: "it fails and returns 404 NOT FOUND when querying a non existent id",
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
			ExpectedStatusCodes: []int{http.StatusNotFound},
			ExpectedResults: []string{`{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
`},

			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/42", nil)
				return req
			},
		},
	}
	getCollectionAsOptionsTestCases = []testCase{
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
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`[
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
]`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options", nil)
				return req
			},
		},
	}
	getCollectionAsOptionsWithParamsTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column" +
				" and returns three records when we filter on purchaseDate",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
			},
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`[
  {
    "id": "3",
    "name": "Buntfink SteelKettle"
  },
  {
    "id": "4",
    "name": "Copper Coffee Pot Cezve"
  }
]`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=&purchaseDate=2017-12-12T12:00:00.000Z", nil)
				return req
			},
		},
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column" +
				" and returns one record when we filter on purchaseDate and provide" +
				" a filter query",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
			},
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`[
  {
    "id": "2",
    "name": "Sanremo Café Racer"
  }
]`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=San&purchaseDate=2017-12-12T12:00:00.123Z", nil)
				return req
			},
		},
	}
	getCollectionAsOptionsFilterableTestCases = []testCase{
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
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`[
  {
    "id": "1",
    "name": "Bialetti Moka Express 6 cup"
  }
]`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=moka", nil)
				return req
			},
		},
	}
)
