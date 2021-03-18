package app

import (
	"fmt"

	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/sql/mysql"
	"github.com/signavio/workflow-connector/internal/pkg/sql/oracle"
	"github.com/signavio/workflow-connector/internal/pkg/sql/postgres"
	"github.com/signavio/workflow-connector/internal/pkg/sql/sqlite"
	"github.com/signavio/workflow-connector/internal/pkg/sql/sqlserver"
)

func NewEndpoint(cfg config.Config) (endpoint.Endpoint, error) {
	switch cfg.Database.Driver {
	case "sqlserver":
		return sqlserver.New(), nil
	case "sqlite3":
		return sqlite.New(), nil
	case "mysql":
		return mysql.New(), nil
	case "postgres":
		return postgres.New(), nil
	case "godror":
		return oracle.New(), nil
	default:
		return nil, fmt.Errorf("Database driver: %s, not supported", cfg.Database.Driver)

	}
}
