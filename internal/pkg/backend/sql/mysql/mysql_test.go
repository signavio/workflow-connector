package mysql

import (
	"fmt"
	"strings"
	"testing"
)

func TestMysql(t *testing.T) {
	if strings.Contains(viper.Get("db").(string), "mysql") &&
		viper.IsSet("db.test.mysql.url") {
		sql.RunTests(t, "mysql", viper.Get("db.test.mysql.url"), NewMysqlBackend)
	} else {
		fmt.Println("tests using mysql test db not performed:\n" +
			"mysql not specified in arg 'db' and/or the setting\n" +
			"'db.test.mysql.url' is not set",
		)
	}
}
