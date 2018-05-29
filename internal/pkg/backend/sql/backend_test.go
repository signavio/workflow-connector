package sql

import (
	"testing"
)

func TestHandlers(t *testing.T) {
	t.Run("GetSingle", func(t *testing.T) {
		for _, tc := range TestCasesGetSingle {
			Run(t, tc)
		}
	})
	t.Run("GetSingleAsOption", func(t *testing.T) {
		for _, tc := range TestCasesGetSingleAsOption {
			Run(t, tc)
		}
	})
	t.Run("GetCollection", func(t *testing.T) {
		for _, tc := range TestCasesGetCollection {
			Run(t, tc)
		}
	})
	t.Run("GetCollectionAsOptions", func(t *testing.T) {
		for _, tc := range TestCasesGetCollectionAsOptions {
			Run(t, tc)
		}
	})
	t.Run("GetCollectionAsOptionsFilterable", func(t *testing.T) {
		for _, tc := range TestCasesGetCollectionAsOptionsFilterable {
			Run(t, tc)
		}
	})
	t.Run("UpdateSingle", func(t *testing.T) {
		for _, tc := range TestCasesUpdateSingle {
			Run(t, tc)
		}
	})
	t.Run("CreateSingle", func(t *testing.T) {
		for _, tc := range TestCasesCreateSingle {
			Run(t, tc)
		}
	})
	t.Run("DeleteSingle", func(t *testing.T) {
		for _, tc := range TestCasesDeleteSingle {
			Run(t, tc)
		}
	})
}
