package sqlite

import (
	"fmt"
	"strings"
	"testing"

	"github.com/signavio/workflow-connector/internal/pkg/backend/sql"
	"github.com/spf13/viper"
)

func TestSqlite(t *testing.T) {
	if strings.Contains(viper.Get("db").(string), "sqlite3") &&
		viper.IsSet("db.test.sqlite3.url") {
		sql.RunTests(t, "sqlite3", viper.Get("db.test.sqlite3.url"), NewSqliteBackend)
	} else {
		fmt.Println("tests using sqlite test db not performed:\n" +
			"sqlite3 not specified in arg 'db' and/or the setting\n" +
			"'db.test.sqlite3.url' is not set",
		)
	}
}
