package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var home = os.Getenv("HOME")

// Order in array is significant, items at the front will precedence
var configPaths = []string{
	// TODO fill out with typical windows config directories
	filepath.Join(home, "/.config/workflow-connector/"),
	"/etc/workflow-connector/",
	"config/",
	"./",
	"../../config",
}

// Config defines the data structures which can be used and configured
// in the config.yaml file and other relevant data structures
type Config struct {
	Port     string
	Database struct {
		Driver string
		URL    string
		Tables []*Table
	}
	TLS struct {
		Enabled    bool
		PublicKey  string
		PrivateKey string
	}
	Descriptor   *Descriptor
	Auth         *Auth
	Logging      string
	TableSchemas map[string]*TableSchema
}

// Table defines the name of the database table that will be queried
// and the table column which will be the `name` field when the
// query results are formatted for option routes.
type Table struct {
	Name               string
	ColumnAsOptionName string
}

// TableSchema defines the schema of the database tables that will be queried
type TableSchema struct {
	ColumnNames []string
	DataTypes   []interface{}
}

// Auth stores the username and password hash. Inbound HTTP request must
// be authenticated over HTTP Basic Auth and the credentials provided
// by the client will be compared to values stored here
type Auth struct {
	Username     string
	PasswordHash string
}

// ContextKey is used as a key when populating a context.Context with values
type ContextKey string

// Initialize configuration file from typical directory locations and parse it
func Initialize(descriptorFile io.Reader) (cfg *Config) {
	viper.SetConfigName("config")
	for _, p := range configPaths {
		viper.AddConfigPath(p)
	}
	viper.AddConfigPath("$HOME/.config/workflow-db-connector/")
	viper.AddConfigPath("/etc/workflow-connector/")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../config/")
	// Nested keys use a single underscore `_` as seperator when
	// imported as environment variables.
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Can not parse config file: %s", err))
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("Unabled to decode config file into struct: %s", err))
	}
	cfg.Descriptor = ParseDescriptorFile(descriptorFile)
	for _, td := range cfg.Descriptor.TypeDescriptors {
		cfg.Database.Tables = append(cfg.Database.Tables,
			&Table{td.TableName, td.ColumnAsOptionName})
	}
	cfg.TableSchemas = make(map[string]*TableSchema)
	return
}

// LocationsForDescriptorFile will look in common directories for the
// descriptor.json file that will be served on the root path for
// inbound HTTP requests. The descriptor.json file is fetched
// parsed by the Workflow Accelerator to determine the schema
// of the data provided by this connector.
func LocationsForDescriptorFile() (descriptorFile *os.File, err error) {
	filename := "descriptor.json"
	makeAbsPath := func(path string) string {
		absPath, _ := filepath.Abs(filepath.Join(path, filename))
		return absPath
	}
	var file *os.File
	var absPaths []string
	for _, p := range configPaths {
		absPath := makeAbsPath(p)
		absPaths = append(absPaths, absPath)
		file, err = os.Open(absPath)
		if err == nil {
			return file, nil
		}
	}
	return nil, fmt.Errorf(
		"couldn't find descriptor.json in these directories: \n%+v",
		absPaths,
	)
}
