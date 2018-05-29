package sqlite

import (
	"fmt"
	"testing"

	"github.com/signavio/workflow-connector/internal/pkg/backend/sql"
	"github.com/spf13/viper"
)

func TestSqlite(t *testing.T) {
	if viper.Get("useRealDB").(bool) {
		sql.RunTests(t, "sqlite3", "../../../../../test.db", NewSqliteBackend)
	} else {
		fmt.Println("tests using sqlite db not performed: arg 'useRealDB' not set to true.")
	}
}
