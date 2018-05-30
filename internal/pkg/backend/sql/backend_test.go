package sql

import (
	"testing"
)

func TestGetSingleHandler(t *testing.T) {
	for _, tc := range TestCasesGetSingle {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestGetAsOptionHandler(t *testing.T) {
	for _, tc := range TestCasesGetSingleAsOption {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestGetCollectionHandler(t *testing.T) {
	for _, tc := range TestCasesGetCollection {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestGetCollectionAsOptionsHandler(t *testing.T) {
	for _, tc := range TestCasesGetCollectionAsOptions {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestGetCollectionAsOptionsFilterableHandler(t *testing.T) {
	for _, tc := range TestCasesGetCollectionAsOptionsFilterable {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestUpdateSingleHandler(t *testing.T) {
	for _, tc := range TestCasesUpdateSingle {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestCreateSingleHandler(t *testing.T) {
	for _, tc := range TestCasesCreateSingle {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}

func TestDeleteSingleHandler(t *testing.T) {
	for _, tc := range TestCasesDeleteSingle {
		t.Run(tc.Name, func(t *testing.T) {
			Run(t, tc)
		})
	}
}
