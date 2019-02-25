package sqltests

import (
	"net/http"
	"net/url"
	"strings"
)

var (
	crudTests = map[string][]testCase{
		"GetSingle":     getSingleTestCases,
		"GetCollection": getCollectionTestCases,
		"CreateSingle":  createSingleTestCases,
		"UpdateSingle":  updateSingleTestCases,
		"DeleteSingle":  deleteSingleTestCases,
	}
	getSingleTestCases = []testCase{
		{
			Kind:                "success",
			Name:                "it succeeds when equipment table contains more than one column",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00.123Z",
  "recipes": [
    {
      "creationDate": "2017-12-13T00:00:00.000Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "%sT00:00:01.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  ]
}`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/2", nil)
				return req
			},
		},
		{

			Kind:                "failure",
			Name:                "it fails and returns 404 NOT FOUND when querying a non existent equipment id",
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
		{
			Kind:                "success",
			Name:                "it succeeds when recipes table contains more than one column",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "creationDate": "2017-12-13T00:00:00.000Z",
  "equipment": {
    "acquisitionCost": {
      "amount": 8477.85,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00.123Z"
  },
  "equipmentId": 2,
  "id": "1",
  "instructions": "do this",
  "lastAccessed": "%sT00:00:01.000Z",
  "lastModified": "2017-12-14T00:00:00.123Z",
  "name": "Espresso single shot"
}`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/recipes/1", nil)
				return req
			},
		},
	}
	getCollectionTestCases = []testCase{
		{
			Kind:                "success",
			Name:                "it succeeds when equipment table contains more than one column",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`[
  {
    "acquisitionCost": {
      "amount": 25.95,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Bialetti Moka Express 6 cup",
    "purchaseDate": "2017-12-11T12:00:00.123Z"
  },
  {
    "acquisitionCost": {
      "amount": 8477.85,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00.123Z"
  },
  {
    "acquisitionCost": {
      "amount": 39.95,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Buntfink SteelKettle",
    "purchaseDate": "2017-12-12T12:00:00.000Z"
  },
  {
    "acquisitionCost": {
      "amount": 49.95,
      "currency": "EUR"
    },
    "id": "4",
    "name": "Copper Coffee Pot Cezve",
    "purchaseDate": "2017-12-12T12:00:00.000Z"
  }
]`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment", nil)
				return req
			},
		},
	}
	createSingleTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when provided with valid parameters as URL parameters",
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 35.99,
    "currency": "EUR"
  },
  "id": "5",
  "name": "French Press",
  "purchaseDate": "2017-04-02T00:00:00.000Z",
  "recipes": []
}`},
			ExpectedStatusCodes: []int{http.StatusCreated, http.StatusNoContent},
			ExpectedHeader: http.Header(map[string][]string{
				"Location": []string{"/equipment/5"},
			}),
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("id", "5")
				postData.Set("name", "French Press")
				postData.Set("acquisitionCost", "35.99")
				postData.Set("purchaseDate", "2017-04-02T00:00:00Z")
				req, _ := http.NewRequest("POST", "/equipment", strings.NewReader(postData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
		},
	}
	updateSingleTestCases = []testCase{
		{
			Kind:                "success",
			Name:                "it succeeds when provided with valid parameters as URL parameters",
			ExpectedStatusCodes: []int{http.StatusOK},
			// purchaseDate is rounded to the nearest second
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 9283.99,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-01T12:34:56.789Z",
  "recipes": [
    {
      "creationDate": "2017-12-13T00:00:00.000Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "%sT00:00:01.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  ]
}`, ".*"},
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("name", "Sanremo Café Racer")
				postData.Set("acquisitionCost", "9283.99")
				postData.Set("purchaseDate", "2017-12-01T12:34:56.789Z")
				req, _ := http.NewRequest("PATCH", "/equipment/2?"+postData.Encode(), nil)
				return req
			},
		},
		{
			Kind:                "success",
			Name:                "it succeeds when user explicitely wants to insert a null value",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": %s,
  "recipes": [
    {
      "creationDate": "2017-12-13T00:00:00.000Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "%sT00:00:01.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  ]
}`, `(null|"0001-01-01T00:00:00.000Z")`, ".*"},
			Request: func() *http.Request {
				body := strings.NewReader(
					`{"name": "Sanremo Café Racer", "acquisitionCost": 8477.85, "purchaseDate": null}`,
				)
				req, _ := http.NewRequest(
					"PATCH",
					"/equipment/2",
					body,
				)
				req.Header.Set("Content-Type", "application/json")
				return req
			},
		},
		{
			Kind:                "success",
			Name:                "it succeeds when provided with valid parameters as json in the request body",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00.123Z",
  "recipes": [
    {
      "creationDate": "2017-12-13T00:00:00.000Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "%sT00:00:01.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  ]
}`, ".*"},
			Request: func() *http.Request {
				body := strings.NewReader(
					`{"name": "Sanremo Café Racer", "acquisitionCost": 8477.85, "purchaseDate": "2017-12-12T12:00:00.123Z"}`,
				)
				req, _ := http.NewRequest(
					"PATCH",
					"/equipment/2",
					body,
				)
				req.Header.Set("Content-Type", "application/json")
				return req
			},
		},
		{

			Kind:                "failure",
			Name:                "it fails and returns 404 NOT FOUND when trying to update a non existent id",
			ExpectedStatusCodes: []int{http.StatusNotFound},
			ExpectedResults: []string{`{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
%s`},
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("name", "Sanremo Café Racer")
				postData.Set("acquisitionCost", "512.23")
				req, _ := http.NewRequest("PATCH", "/equipment/42?"+postData.Encode(), nil)
				return req
			},
		},
	}
	deleteSingleTestCases = []testCase{
		{
			Kind:                "success",
			Name:                "it succeeds in deleting an existing resource",
			ExpectedStatusCodes: []int{http.StatusOK},
			ExpectedResults: []string{`{
  "status": {
    "code": 200,
    "description": "Resource with uniqueID '5' successfully deleted from equipment table"
  }
}`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("DELETE", "/equipment/5", nil)
				return req
			},
		},
		{

			Kind:                "failure",
			Name:                "it fails and returns 404 NOT FOUND when trying to delete a non existent id",
			ExpectedStatusCodes: []int{http.StatusNotFound},
			ExpectedResults: []string{`{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
`},
			Request: func() *http.Request {
				req, _ := http.NewRequest("DELETE", "/equipment/42", nil)
				return req
			},
		},
	}
)
