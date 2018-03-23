package app

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/signavio/workflow-connector/pkg/config"
)

type testCase struct {
	name            string
	route           string
	expectedResults string
	postData        url.Values
	request         *http.Request
}

var queryInitializeDB = "CREATE TABLE IF NOT EXISTS equipment (" +
	"id integer not null primary key, " +
	"name text, " +
	"acquisition_cost real, " +
	"purchase_date datetime); " +
	"INSERT INTO equipment(id, name, acquisition_cost, purchase_date) " +
	"VALUES " +
	"(1,'Stainless Steel Cooling Spiral',119.0,'2017-09-07 12:00:00'), " +
	"(2,'Fermentation Tank (50L)',250.0,'2014-09-07 11:00:00'), " +
	"(3,'Temperature Gauge',49.99,'2017-09-04 11:00:00'), " +
	"(4,'Masch Tun (50L)',199.99,'2016-09-04 11:00:00'), " +
	"(5,'Malt mill 550',1270,'2016-09-04 11:00:00'); " +

	"CREATE TABLE IF NOT EXISTS maintenance (" +
	"id integer not null primary key, " +
	"equipment_id integer not null, " +
	"maintenance_performed datetime, " +
	"notes text, " +
	"next_maintenance datetime); " +
	"INSERT INTO maintenance(id, equipment_id, maintenance_performed, notes, next_maintenance) " +
	"VALUES " +
	"(1,3,'2017-02-07 12:00:00','Nothing noteworthy 1','2018-12-01 12:00:00'), " +
	"(2,2,'2015-02-07 12:00:00','Nothing noteworthy 2','2016-11-01 12:00:00'), " +
	"(3,3,'2017-02-07 12:00:00','Nothing noteworthy 3','2018-11-01 12:00:00'), " +
	"(4,1,'2017-02-07 12:00:00','Nothing noteworthy 4','2018-11-01 12:00:00'), " +
	"(5,2,'2016-02-07 12:00:00','Nothing noteworthy 5','2017-11-01 12:00:00'), " +
	"(6,2,'2017-02-07 12:00:00','Nothing noteworthy 6','2018-11-01 12:00:00'); " +

	"CREATE TABLE IF NOT EXISTS warranty (" +
	"id integer not null primary key, " +
	"equipment_id integer not null, " +
	"start_date datetime, " +
	"duration_in_years number, " +
	"conditions text); " +
	"INSERT INTO warranty(id, equipment_id, start_date, duration_in_years, conditions) " +
	"VALUES " +
	"(1,1,'2016-02-20 12:00:00',3,'warranty covers parts and labour'), " +
	"(2,2,'2016-10-02 12:00:00',2,'warranty only for parts'), " +
	"(3,3,'2017-02-19 12:00:00',3,'warranty covers parts and labour'), " +
	"(4,5,'2017-02-19 12:00:00',2,'warranty only for parts'); "

var testCasesForGetSingleEquipment = func(caseType string) []testCase {
	successCases := []testCase{
		{
			name:  "equipment_id=1 with denormalized relationships",
			route: "/equipment/1",
			expectedResults: `{
  "cost": {
    "amount": 119,
    "currency": "EUR"
  },
  "equipmentMaintenance": [
    {
      "equipmentId": 1,
      "id": "4",
      "maintenancePerformed": "2017-02-07T12:00:00Z",
      "nextMaintenance": "2018-11-01T12:00:00Z",
      "notes": "Nothing noteworthy 4"
    }
  ],
  "equipmentWarranty": [
    {
      "conditions": "warranty covers parts and labour",
      "durationInYears": 3,
      "equipmentId": 1,
      "id": "1",
      "startDate": "2016-02-20T12:00:00Z"
    }
  ],
  "id": "1",
  "name": "Stainless Steel Cooling Spiral",
  "purchaseDate": "2017-09-07T12:00:00Z"
}`,
		},
		{
			name:  "equipment_id=2 with denormalized relationships",
			route: "/equipment/2",
			expectedResults: `{
  "cost": {
    "amount": 250,
    "currency": "EUR"
  },
  "equipmentMaintenance": [
    {
      "equipmentId": 2,
      "id": "2",
      "maintenancePerformed": "2015-02-07T12:00:00Z",
      "nextMaintenance": "2016-11-01T12:00:00Z",
      "notes": "Nothing noteworthy 2"
    },
    {
      "equipmentId": 2,
      "id": "5",
      "maintenancePerformed": "2016-02-07T12:00:00Z",
      "nextMaintenance": "2017-11-01T12:00:00Z",
      "notes": "Nothing noteworthy 5"
    },
    {
      "equipmentId": 2,
      "id": "6",
      "maintenancePerformed": "2017-02-07T12:00:00Z",
      "nextMaintenance": "2018-11-01T12:00:00Z",
      "notes": "Nothing noteworthy 6"
    }
  ],
  "equipmentWarranty": [
    {
      "conditions": "warranty only for parts",
      "durationInYears": 2,
      "equipmentId": 2,
      "id": "2",
      "startDate": "2016-10-02T12:00:00Z"
    }
  ],
  "id": "2",
  "name": "Fermentation Tank (50L)",
  "purchaseDate": "2014-09-07T11:00:00Z"
}`,
		},
		{
			name:  "equipment_id=3 with denormalized relationships",
			route: "/equipment/3",
			expectedResults: `{
  "cost": {
    "amount": 49.99,
    "currency": "EUR"
  },
  "equipmentMaintenance": [
    {
      "equipmentId": 3,
      "id": "1",
      "maintenancePerformed": "2017-02-07T12:00:00Z",
      "nextMaintenance": "2018-12-01T12:00:00Z",
      "notes": "Nothing noteworthy 1"
    },
    {
      "equipmentId": 3,
      "id": "3",
      "maintenancePerformed": "2017-02-07T12:00:00Z",
      "nextMaintenance": "2018-11-01T12:00:00Z",
      "notes": "Nothing noteworthy 3"
    }
  ],
  "equipmentWarranty": [
    {
      "conditions": "warranty covers parts and labour",
      "durationInYears": 3,
      "equipmentId": 3,
      "id": "3",
      "startDate": "2017-02-19T12:00:00Z"
    }
  ],
  "id": "3",
  "name": "Temperature Gauge",
  "purchaseDate": "2017-09-04T11:00:00Z"
}`,
		},
		{
			name:  "equipment_id=4 with denormalized relationships",
			route: "/equipment/4",
			expectedResults: `{
  "cost": {
    "amount": 199.99,
    "currency": "EUR"
  },
  "equipmentMaintenance": [],
  "equipmentWarranty": [],
  "id": "4",
  "name": "Masch Tun (50L)",
  "purchaseDate": "2016-09-04T11:00:00Z"
}`,
		},
		{
			name:  "equipment_id=5 with denormalized relationships",
			route: "/equipment/5",
			expectedResults: `{
  "cost": {
    "amount": 1270,
    "currency": "EUR"
  },
  "equipmentMaintenance": [],
  "equipmentWarranty": [
    {
      "conditions": "warranty only for parts",
      "durationInYears": 2,
      "equipmentId": 5,
      "id": "4",
      "startDate": "2017-02-19T12:00:00Z"
    }
  ],
  "id": "5",
  "name": "Malt mill 550",
  "purchaseDate": "2016-09-04T11:00:00Z"
}`,
		},
		{
			name:            "equipment_id=999 (non existent)",
			route:           "/equipment/999",
			expectedResults: `{}`,
		},
	}
	failureCases := []testCase{}
	if caseType == "success" {
		return successCases
	}
	return failureCases
}
var testCasesForGetSingleEquipmentOption = func(caseType string) []testCase {
	successCases := []testCase{
		{
			name:  "equipment table column `id` and `name`",
			route: "/equipment/options/1",
			expectedResults: `{
  "id": "1",
  "name": "Stainless Steel Cooling Spiral"
}`,
		},
		{
			name:            "equipment table column `id` and `name`",
			route:           "/equipment/options/999",
			expectedResults: `{}`,
		},
	}
	failureCases := []testCase{}
	if caseType == "success" {
		return successCases
	}
	return failureCases
}
var testCasesForGetCollectionEquipment = func(caseType string) []testCase {
	successCases := []testCase{
		{
			name:  "equipment table (collection) without relationships",
			route: "/equipment",
			expectedResults: `[
  {
    "cost": {
      "amount": 119,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Stainless Steel Cooling Spiral",
    "purchaseDate": "2017-09-07T12:00:00Z"
  },
  {
    "cost": {
      "amount": 250,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Fermentation Tank (50L)",
    "purchaseDate": "2014-09-07T11:00:00Z"
  },
  {
    "cost": {
      "amount": 49.99,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Temperature Gauge",
    "purchaseDate": "2017-09-04T11:00:00Z"
  },
  {
    "cost": {
      "amount": 199.99,
      "currency": "EUR"
    },
    "id": "4",
    "name": "Masch Tun (50L)",
    "purchaseDate": "2016-09-04T11:00:00Z"
  },
  {
    "cost": {
      "amount": 1270,
      "currency": "EUR"
    },
    "id": "5",
    "name": "Malt mill 550",
    "purchaseDate": "2016-09-04T11:00:00Z"
  }
]`,
		},
	}
	failureCases := []testCase{}
	if caseType == "success" {
		return successCases
	}
	return failureCases
}

var testCasesForGetCollectionEquipmentAsOptions = func(caseType string) []testCase {
	successCases := []testCase{
		{
			name:  "equipment table (collection) as options",
			route: "/equipment/options",
			expectedResults: `[
  {
    "id": "1",
    "name": "Stainless Steel Cooling Spiral"
  },
  {
    "id": "2",
    "name": "Fermentation Tank (50L)"
  },
  {
    "id": "3",
    "name": "Temperature Gauge"
  },
  {
    "id": "4",
    "name": "Masch Tun (50L)"
  },
  {
    "id": "5",
    "name": "Malt mill 550"
  }
]`,
		},
	}
	failureCases := []testCase{}
	if caseType == "success" {
		return successCases
	}
	return failureCases
}

var testCasesForGetCollectionEquipmentAsOptionsFilterable = func(caseType string) []testCase {
	successCases := []testCase{
		{
			name:  "filtered (using string `ta`) equipment table collection as options",
			route: "/equipment/options?filter=ta",
			expectedResults: `[
  {
    "id": "1",
    "name": "Stainless Steel Cooling Spiral"
  },
  {
    "id": "2",
    "name": "Fermentation Tank (50L)"
  }
]`,
		},
		{
			name:            "filtered (using string `juiklo`) equipment table collection as options",
			route:           "/equipment/options?filter=juiklo",
			expectedResults: `[{}]`,
		},
	}
	failureCases := []testCase{}
	if caseType == "success" {
		return successCases
	}
	return failureCases
}

func initializeSqliteDB(cfg *config.Config, t *testing.T) {
	db, err := sql.Open(
		cfg.Database.Driver,
		cfg.Database.URL,
	)
	if err != nil {
		t.Errorf("Error opening connection to database: %s", err)
	}
	_, err = db.Exec(queryInitializeDB)
	if err != nil {
		fmt.Printf("Test tables probably already exist, here is the error anyway: %s\n", err)
	}
}

func setupEndToEndTestsWithSqliteBackend(t *testing.T) (app *App) {
	withDescriptor, err := os.Open("../../config/descriptor.json")
	if err != nil {
		t.Errorf("unable to open config file: %v", err)
	}
	cfg := config.Initialize(withDescriptor)
	initializeSqliteDB(cfg, t)
	app = NewApp(cfg)
	app.DefineRoutes()
	return
}

func commonTest(testCases func(string) []testCase, testName, method, route string, t *testing.T) {
	t.Run(testName, func(t *testing.T) {
		for _, tc := range testCases("success") {
			app := setupEndToEndTestsWithSqliteBackend(t)
			ts := httptest.NewServer(app.Server.Handler)
			defer ts.Close()
			req, err := http.NewRequest(method, ts.URL+tc.route, nil)
			if err != nil {
				t.Errorf("Expected no error, instead we received: %s", err)
			}
			req.SetBasicAuth(app.Cfg.Auth.Username, "Foobar")
			client := ts.Client()
			response, err := client.Do(req)
			if err != nil {
				t.Errorf("Expected no error, instead we received: %s", err)
			}
			got, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Errorf("Expected no error, instead we received: %s", err)
			}
			if response.StatusCode != 200 {
				t.Errorf("Expected no error, instead we received: %s", err)
			}
			if string(got[:]) != tc.expectedResults {
				t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
					got, tc.expectedResults)
			}
		}
	})
}

func TestGetSingleEquipment(t *testing.T) {
	commonTest(
		testCasesForGetSingleEquipment,
		"returns a single Equipment",
		"GET",
		"/equipment/1",
		t,
	)
}

func TestGetSingleEquipmentOption(t *testing.T) {
	commonTest(
		testCasesForGetSingleEquipmentOption,
		"Returns a single equipment as option",
		"GET",
		"/equipment/options/1",
		t,
	)
}

func TestGetCollectionEquipment(t *testing.T) {
	commonTest(
		testCasesForGetCollectionEquipment,
		"Returns a collection of equipments",
		"GET",
		"/equipment",
		t,
	)
}

func TestGetCollectionEquipmentAsOptions(t *testing.T) {
	commonTest(
		testCasesForGetCollectionEquipmentAsOptions,
		"Returns a collection of equipments as options",
		"GET",
		"/equipment/options",
		t,
	)
}

func TestGetCollectionEquipmentAsOptionsFilterable(t *testing.T) {
	commonTest(
		testCasesForGetCollectionEquipmentAsOptionsFilterable,
		"Returns a filtered collection of equipments as options",
		"GET",
		"/equipment/options?filter=ta",
		t,
	)
}
