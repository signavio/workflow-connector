package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Order is significant, items at the front of the array take precedence
var configPaths = []string{
	// TODO fill out with typical windows config directories
	filepath.Join(os.Getenv("HOME"), "/.config/workflow-connector/"),
	"/etc/workflow-connector/",
	"config/",
	"./",
	os.Getenv("WORKFLOW_CONFIG"),
}

// Options is populated by the packages init() function
// TODO It should be a singleton
var Options Config

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
	Descriptor *Descriptor
	Auth       *Auth
	Logging    string
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

// useRealDB is a command line flag that will make the tests run on the database
// specified in the config file, instead of using sqlmock
type useRealDB struct {
	name string
	val  bool
}

// Initialize configuration file from typical directory locations and parse it
func init() {
	viper.SetConfigName("config")
	for _, p := range configPaths {
		viper.AddConfigPath(p)
	}
	// Nested keys use a single underscore `_` as seperator when
	// imported as environment variables.
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Can not parse config file: %s", err))
	}
	if err := viper.Unmarshal(&Options); err != nil {
		panic(fmt.Errorf("Unabled to decode config file into struct: %s", err))
	}
	useRealDB := &useRealDB{name: "useRealDB", val: false}
	flag.BoolVar(&useRealDB.val, "useRealDB", false, "run tests on the real DB")
	viper.BindFlagValue("useRealDB", useRealDB)
	flag.Parse()
	descriptorFile, err := locationsForDescriptorFile()
	if err != nil {
		panic(err)
	}
	Options.Descriptor = ParseDescriptorFile(descriptorFile)
	for _, td := range Options.Descriptor.TypeDescriptors {
		Options.Database.Tables = append(Options.Database.Tables,
			&Table{td.TableName, td.ColumnAsOptionName})
	}
}

// locationsForDescriptorFile will look in common directories for the
// descriptor.json file that will be served on the root path for
// inbound HTTP requests. The descriptor.json file is parsed
// by the Workflow Accelerator to determine the schema
// of the data provided by this connector.
func locationsForDescriptorFile() (descriptorFile *os.File, err error) {
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
func (f useRealDB) HasChanged() bool    { return false }
func (f useRealDB) Name() string        { return f.name }
func (f useRealDB) ValueString() string { return fmt.Sprintf("%v", f.val) }
func (f useRealDB) ValueType() string   { return "bool" }
