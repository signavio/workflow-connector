package sqltests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/signavio/workflow-connector/internal/pkg/sql/mysql"
	"github.com/signavio/workflow-connector/internal/pkg/sql/oracle"
	"github.com/signavio/workflow-connector/internal/pkg/sql/postgres"
	"github.com/signavio/workflow-connector/internal/pkg/sql/sqlserver"
	"github.com/spf13/viper"
)

var queryTemplates = map[string]string{
	"GetSingle": "SELECT * " +
		"  FROM {{.TableName}} AS _{{.TableName}} " +
		"  {{range .Relations}}" +
		"     LEFT JOIN {{.Relationship.WithTable}}" +
		"     ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
		"     = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}" +
		"  {{end}}" +
		"  WHERE _{{$.TableName}}.{{$.UniqueIDColumn}} = ?",
	"GetSingleAsOption": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
		"FROM {{.TableName}} " +
		"WHERE {{.UniqueIDColumn}} = ?",
	"GetCollection": "SELECT * " +
		"FROM {{.TableName}}",
	"GetCollectionFilterable": "SELECT * " +
		"FROM {{.TableName}} " +
		"WHERE {{.FilterOnColumn}} {{.Operator}} ?",
	"GetCollectionAsOptions": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
		"FROM {{.TableName}}",
	"GetCollectionAsOptionsFilterable": "SELECT {{.UniqueIDColumn}}, {{.ColumnAsOptionName}} " +
		"FROM {{.TableName}} " +
		"WHERE {{.ColumnAsOptionName}} LIKE ?",
	"UpdateSingle": "UPDATE {{.TableName}} SET {{.ColumnNames | head}}" +
		" = ?{{range .ColumnNames | tail}}, {{.}} = ?{{end}} WHERE {{.UniqueIDColumn}} = ?",
	"CreateSingle": "INSERT INTO {{.TableName}}({{.ColumnNames | head}}" +
		"{{range .ColumnNames | tail}}, {{.}}{{end}}) " +
		"VALUES(?{{range .ColumnNames | tail}}, ?{{end}})",
	"DeleteSingle": "DELETE FROM {{.TableName}} WHERE {{.UniqueIDColumn}} = ?",
	"GetTableSchema": "SELECT * " +
		"FROM {{.TableName}} " +
		"LIMIT 1",
	"GetTableWithRelationshipsSchema": "SELECT * " +
		"FROM {{.TableName}} AS _{{.TableName}}" +
		"{{range .Relations}}" +
		" LEFT JOIN {{.Relationship.WithTable}}" +
		" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignTableUniqueIDColumn}}" +
		" = _{{$.TableName}}.{{.Relationship.LocalTableUniqueIDColumn}}{{end}} LIMIT 1",
}

// TestCase for sql backend
type testCase struct {
	// A testCase should assert success cases or failure cases
	Kind string
	// A testCase has a unique name
	Name string
	// A testCase has descriptor fields that describe the schema of the
	// mocked database table in workflow accelerator's custom json format
	ExpectedResults []string
	// A testCase contains the expected http status code(s) that should be
	// returned to the client
	ExpectedStatusCodes []int
	// A testCase contains the expected key-value pairs present in the http
	// header that is returned to the client
	ExpectedHeader http.Header
	// A testCase contains the test data that a client would submit in an
	// HTTP POST
	PostData url.Values
	// A testCase contains a *http.Request
	Request func() *http.Request
	// run the testcase
	Run func(tc testCase, ts *httptest.Server) error
}

func TestSqlBackends(t *testing.T) {
	var testUsingDB string
	if viper.IsSet("db") {
		testUsingDB = viper.Get("db").(string)
	}
	switch {
	case strings.Contains(testUsingDB, "mysql"):
		testSqlBackend(t, "mysql", "mysql", mysql.New)
	case strings.Contains(testUsingDB, "oracle"):
		testSqlBackend(t, "oracle", "godror", oracle.New)
	case strings.Contains(testUsingDB, "sqlserver"):
		testSqlBackend(t, "sqlserver", "sqlserver", sqlserver.New)
	default:
		testSqlBackend(t, "postgres", "postgres", postgres.New)
	}
}

func testSqlBackend(t *testing.T, name, driver string, newEndpointFunc func() endpoint.Endpoint) {
	endpoint := newEndpointFunc()
	err := endpoint.Open(
		driver,
		viper.Get(name+".database.url").(string),
	)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	t.Run("Using "+name+" database", func(t *testing.T) {
		ts := newTestServer(endpoint)
		defer ts.Close()
		for testName, testCases := range conformityTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range crudTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range dataConnectorOptionsTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}
		for testName, testCases := range collectionFiltererTests {
			runTestCases(t, testName, testCases, ts, endpoint)
		}

	})
}
func runTestCases(t *testing.T, testName string, testCases []testCase, ts *httptest.Server, endpoint endpoint.Endpoint) {
	t.Run(testName, func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				err := run(tc, ts)
				if err != nil {
					t.Errorf(err.Error())
					return
				}
			})
		}
	})
}

func run(tc testCase, ts *httptest.Server) error {
	switch tc.Kind {
	case "success":
		tc.Run = itSucceeds
		if err := tc.Run(tc, ts); err != nil {
			return err
		}
		return nil
	case "failure":
		tc.Run = itFails
		if err := tc.Run(tc, ts); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("testcase should either be success or failure kind")
	}
}

func itFails(tc testCase, ts *httptest.Server) error {
	req := tc.Request()
	u, err := url.Parse(ts.URL + req.URL.RequestURI())
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	req.URL = u
	req.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	if !in(tc.ExpectedStatusCodes, res.StatusCode) {
		return fmt.Errorf(
			"expected one of HTTP %+v, instead we received: %d",
			tc.ExpectedStatusCodes,
			res.StatusCode,
		)
	}
	if !match(string(got[:]), tc.ExpectedResults[0], tc.ExpectedResults[1:]...) {
		return fmt.Errorf(
			"response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s",
			got,
			interpolateRegexp(tc.ExpectedResults[0], tc.ExpectedResults[1:]...),
		)
	}
	return nil
}

func itSucceeds(tc testCase, ts *httptest.Server) error {
	req := tc.Request()
	u, err := url.Parse(ts.URL + req.URL.RequestURI())
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	req.URL = u
	req.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}

	got, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	if !in(tc.ExpectedStatusCodes, res.StatusCode) {
		return fmt.Errorf(
			"expected one of HTTP %+v, instead we received: %d",
			tc.ExpectedStatusCodes,
			res.StatusCode,
		)
	}
	if tc.ExpectedHeader != nil {
		if res.Header.Get("Location") == tc.ExpectedHeader.Get("Location") {
			return fmt.Errorf(
				"expected HTTP Header %s, instead we received: %s",
				res.Header.Get("Location"),
				tc.ExpectedHeader.Get("Location"),
			)
		}
	}
	if !match(string(got[:]), tc.ExpectedResults[0], tc.ExpectedResults[1:]...) {
		return fmt.Errorf(
			"response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s",
			got,
			interpolateRegexp(tc.ExpectedResults[0], tc.ExpectedResults[1:]...),
		)
	}
	return nil
}
func newTestServer(e endpoint.Endpoint) *httptest.Server {
	router := e.GetHandler().(*mux.Router)
	ts := httptest.NewUnstartedServer(router)
	router.Use(middleware.BasicAuth)
	router.Use(middleware.RouteChecker)
	router.Use(middleware.RequestInjector)
	router.Use(middleware.ResponseInjector)
	server := &http.Server{}
	server.Handler = router
	ts.Config = server
	ts.Start()
	return ts
}

func match(got, expected string, regexps ...string) (matched bool) {
	expectedWithRegexp := interpolateRegexp(expected, regexps...)
	matched, err := regexp.MatchString(expectedWithRegexp, got)
	if err != nil {
		panic(err)
	}
	return
}

func interpolateRegexp(expected string, regexps ...string) (interpolatedRegexp string) {
	var regexpsToUse []interface{}
	for _, regexp := range regexps {
		regexpsToUse = append(regexpsToUse, regexp)
	}
	quoteUnintentionalMetacharacters := regexp.QuoteMeta(expected)
	interpolatedRegexp = quoteUnintentionalMetacharacters
	if len(regexps) > 0 {
		interpolatedRegexp = fmt.Sprintf(
			quoteUnintentionalMetacharacters,
			regexpsToUse...,
		)
	}
	return
}
func in(statusCodes []int, a int) (result bool) {
	for _, statusCode := range statusCodes {
		if a == statusCode {
			result = true
		}
	}
	return
}
