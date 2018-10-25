package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/spf13/viper"
)

// Options is populated by this package's init() function
// TODO It should be a singleton
var Options Config

// Config defines the data structures which can be used and configured
// in the config.yaml file and other relevant data structures
type Config struct {
	Port     string
	Endpoint struct {
		Driver string
		URL    string
		Tables []*Table
	}
	TLS struct {
		Enabled    bool
		PublicKey  string
		PrivateKey string
	}
	Descriptor *Descriptor
	Auth       *Auth
	Logging    bool
}

// Table defines the name of the database table that will be queried
// and the table column which will be the `name` field when the
// query results are formatted for option routes.
type Table struct {
	Name               string
	ColumnAsOptionName string
}

// Auth stores the username and password hash. Inbound HTTP request must
// be authenticated over HTTP Basic Auth and the credentials provided
// by the client will be compared to values stored here
type Auth struct {
	Username     string
	PasswordHash string
}

// db is a command line flag that takes a comma seperated list of databases to test
type db struct {
	name string
	val  string
}

// config is a command line flag that specifies the path to the config directory
type configDir struct {
	name string
	val  string
}

// Initialize configuration file from typical directory locations and parse it
func init() {
	db := &db{name: "db", val: ""}
	flag.StringVar(&db.val, "db", "", "run tests on the real test databases")
	viper.BindFlagValue("db", db)
	configDir := &configDir{name: "config-dir", val: "config"}
	flag.StringVar(&configDir.val, "config-dir", "", "specify location to config directory")
	viper.BindFlagValue("configDir", configDir)
	flag.Parse()
	viper.SetConfigName("config")
	if configDir.ValueString() == "" {
		viper.AddConfigPath("config")
		viper.AddConfigPath(filepath.Join("../../../", "config"))
		viper.AddConfigPath(filepath.Join("../../../../", "config"))
	} else {
		viper.AddConfigPath(configDir.ValueString())
	}
	viper.AutomaticEnv()
	// Nested keys use a single underscore `_` as seperator when
	// imported as environment variables.
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Can not parse config file: %v\n", err)
	}
	if err := viper.Unmarshal(&Options); err != nil {
		log.Fatalf("Unable to decode config file into struct: %s", err)
	}
	descriptorFile, err := os.Open(descriptorFilePath())
	if err != nil {
		log.Fatalf("Unable to open descriptor.json file: %v\n", err)
	}
	Options.Descriptor = ParseDescriptorFile(descriptorFile)
	for _, td := range Options.Descriptor.TypeDescriptors {
		Options.Endpoint.Tables = append(Options.Endpoint.Tables,
			&Table{td.TableName, td.ColumnAsOptionName})
	}
}
func descriptorFilePath() string {
	configFile := viper.ConfigFileUsed()
	configDir := filepath.Dir(configFile)
	return filepath.Join(configDir, "descriptor.json")
}

func (f db) HasChanged() bool           { return false }
func (f db) Name() string               { return f.name }
func (f db) ValueString() string        { return f.val }
func (f db) ValueType() string          { return "string" }
func (f configDir) HasChanged() bool    { return false }
func (f configDir) Name() string        { return f.name }
func (f configDir) ValueString() string { return f.val }
func (f configDir) ValueType() string   { return "string" }
