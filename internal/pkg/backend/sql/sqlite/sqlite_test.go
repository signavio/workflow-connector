package sqlite

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/backend/sql"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/spf13/viper"
)

func itSucceeds(t *testing.T, tc sql.TestCase) {
	backend, err := setupBackendWithRealDB()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	err = backend.Open(
		config.Options.Database.Driver,
		"../../../../../test.db",
	)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	ts := setupTestServer(backend)
	defer ts.Close()
	tc.Request.URL, err = url.Parse(ts.URL + tc.Request.URL.String())
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	tc.Request.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(tc.Request)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	if strings.HasPrefix(string(res.StatusCode), "2") {
		t.Errorf("Expected HTTP 2xx, instead we received: %d", res.StatusCode)
	}
	if string(got[:]) != tc.ExpectedResults {
		t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
			got, tc.ExpectedResults)
	}
}

func itFails(t *testing.T, tc sql.TestCase) {
	backend, err := setupBackendWithRealDB()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	err = backend.Open(
		config.Options.Database.Driver,
		"../../../../../test.db",
	)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	ts := setupTestServer(backend)
	defer ts.Close()
	tc.Request.URL, err = url.Parse(ts.URL + tc.Request.URL.String())
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	tc.Request.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(tc.Request)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("Expected 404 Not Found, instead we received: %v", res.StatusCode)
	}
	if string(got[:]) != tc.ExpectedResults {
		t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
			got, tc.ExpectedResults)
	}
}

func setupBackendWithRealDB() (b *sql.Backend, err error) {
	return NewSqliteBackend(), nil
}

func setupTestServer(b *sql.Backend) *httptest.Server {
	router := b.GetHandler().(*mux.Router)
	ts := httptest.NewUnstartedServer(router)
	router.Use(middleware.BasicAuth)
	router.Use(middleware.RequestInjector)
	router.Use(middleware.ResponseInjector)
	server := &http.Server{}
	server.Handler = router
	ts.Config = server
	ts.Start()
	return ts
}

func TestSqlite(t *testing.T) {
	if viper.Get("useRealDB").(bool) {
		do(t)
	} else {
		fmt.Println("tests using sqlite db not performed: arg 'useRealDB' not set to true.")
	}
}

func do(t *testing.T) {
	t.Run("GetSingle", func(t *testing.T) {
		for _, tc := range sql.TestCasesGetSingle {
			run(tc, t)
		}
	})
	t.Run("GetSingleAsOption", func(t *testing.T) {
		for _, tc := range sql.TestCasesGetSingleAsOption {
			run(tc, t)
		}
	})
	t.Run("GetCollection", func(t *testing.T) {
		for _, tc := range sql.TestCasesGetCollection {
			run(tc, t)
		}
	})
	t.Run("GetCollectionAsOptions", func(t *testing.T) {
		for _, tc := range sql.TestCasesGetCollectionAsOptions {
			run(tc, t)
		}
	})
	t.Run("GetCollectionAsOptionsFilterable", func(t *testing.T) {
		for _, tc := range sql.TestCasesGetCollectionAsOptionsFilterable {
			run(tc, t)
		}
	})
	t.Run("UpdateSingle", func(t *testing.T) {
		for _, tc := range sql.TestCasesUpdateSingle {
			run(tc, t)
		}
	})
	t.Run("CreateSingle", func(t *testing.T) {
		for _, tc := range sql.TestCasesCreateSingle {
			run(tc, t)
		}
	})
	t.Run("DeleteSingle", func(t *testing.T) {
		for _, tc := range sql.TestCasesDeleteSingle {
			run(tc, t)
		}
	})

}

func run(tc sql.TestCase, t *testing.T) {
	t.Run(tc.Name, func(t *testing.T) {
		if tc.Kind == "success" {
			tc.Run = itSucceeds
			tc.Run(t, tc)
		} else if tc.Kind == "failure" {
			tc.Run = itFails
			tc.Run(t, tc)
		} else {
			t.Errorf("testcase should either be success or failure kind")
		}
	})
}
