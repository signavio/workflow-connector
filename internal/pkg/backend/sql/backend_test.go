package sql



func itFails(t *testing.T, tc TestCase) {
	// The config.Descriptor in config.Options needs to be mocked
	mockedDescriptorFile, err := mockDescriptorFile(tc.descriptorFields)
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	config.Options.Descriptor = config.ParseDescriptorFile(mockedDescriptorFile)
	backend, mock, err := setupBackendWithMockedDB()
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	// initialize mock database
	tc.expectedQueries(mock, tc.columnNames, tc.rowsAsCsv)
	// mock the database table schema
	backend.TableSchemas["equipment"] = tc.tableSchema
	ts := setupTestServer(backend)
	defer ts.Close()
	tc.request.URL, err = url.Parse(ts.URL + tc.request.URL.String())
	if err != nil {
		t.Errorf("Expected no error, instead we received: %s", err)
	}
	tc.request.SetBasicAuth(config.Options.Auth.Username, "Foobar")
	client := ts.Client()
	res, err := client.Do(tc.request)
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
	if string(got[:]) != tc.expectedResults {
		t.Errorf("Response doesn't match what we expected\nResponse:\n%s\nExpected:\n%s\n",
			got, tc.expectedResults)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func setupBackendWithMockedDB() (b *Backend, mock sqlmock.Sqlmock, err error) {
	b = NewBackend()
	b.Templates = queryTemplates
	b.DB, mock, err = sqlmock.New()
	if err != nil {
		return nil, mock, fmt.Errorf(
			"an error '%s' was not expected when opening a stub database connection",
			err,
		)
	}
	return
}
func setupTestServer(b *Backend) *httptest.Server {
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
func mockDescriptorFile(testCaseDescriptorFields string) (io.Reader, error) {
	mockedDescriptorFile := fmt.Sprintf(
		descriptorFileBase,
		testCaseDescriptorFields,
	)
	return strings.NewReader(mockedDescriptorFile), nil
}
