package descriptor

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

type Descriptor struct {
	Key             string            `json:"key,omitempty"`
	Name            string            `json:"name,omitempty"`
	Description     string            `json:"description,omitempty"`
	TypeDescriptors []*TypeDescriptor `json:"typeDescriptors,omitempty"`
	Version         int               `json:"version,omitempty"`
	ProtocolVersion int               `json:"protocolVersion,omitempty"`
}

type TypeDescriptor struct {
	Key                string   `json:"key,omitempty"`
	Name               string   `json:"name,omitempty"`
	TableName          string   `json:"tableName,omitempty"`
	ColumnAsOptionName string   `json:"columnAsOptionName,omitempty"`
	UniqueIdColumn     string   `json:"uniqueIdColumn,omitempty"`
	RecordType         string   `json:"recordType,omitempty"`
	Fields             []*Field `json:"fields,omitempty"`
	OptionsAvailable   bool     `json:"optionsAvailable,omitempty"`
	FetchOneAvailable  bool     `json:"fetchOneAvailable,omitempty"`
}

type Field struct {
	Key          string        `json:"key,omitempty"`
	Name         string        `json:"name,omitempty"`
	Type         *WorkflowType `json:"type,omitempty"`
	FromColumn   string        `json:"fromColumn,omitempty"`
	Relationship *Relationship `json:"relationship,omitempty"`
}

type WorkflowType struct {
	Name     string    `json:"name,omitempty"`
	Kind     string    `json:"kind,omitempty"`
	Amount   *Amount   `json:"amount,omitempty"`
	Currency *Currency `json:"currency,omitempty"`
}
type Amount struct {
	Key        string `json:"key,omitempty"`
	FromColumn string `json:"fromColumn,omitempty"`
}

type Currency struct {
	Key        string `json:"key,omitempty"`
	FromColumn string `json:"fromColumn,omitempty"`
	Value      string `json:"value,omitempty"`
}

type Relationship struct {
	Kind                       string `json:"kind,omitempty"`
	WithTable                  string `json:"withTable,omitempty"`
	ForeignTableUniqueIDColumn string `json:"foreignTableUniqueIdColumn,omitempty"`
	LocalTableUniqueIDColumn   string `json:"localTableUniqueIdColumn,omitempty"`
}

// SchemaMapping defines the schema of data retrieved from a particular backend
type SchemaMapping struct {
	FieldNames    []string
	BackendTypes  []interface{}
	GolangTypes   []interface{}
	WorkflowTypes []interface{}
}

// ParseDescriptorFile will parse the descriptor.json file and make sure
// to add an `id` field if the user has not already specified it
func ParseDescriptorFile(file io.Reader) (descriptor *Descriptor) {
	var content []byte
	content, err := ioutil.ReadAll(file)
	if err != nil {
		panic(fmt.Errorf("Unable to read descriptor.json file: %v", err))
	}
	err = json.Unmarshal(content, &descriptor)
	if err != nil {
		panic(fmt.Errorf("Unable to unmarshal descriptor.json: %v", err))
	}
	descriptor = addIDFieldIfNotExists(descriptor)
	if err := performSanityChecks(descriptor); err != nil {
		panic(err)
	}
	return
}

func addIDFieldIfNotExists(descriptor *Descriptor) *Descriptor {
	for _, td := range descriptor.TypeDescriptors {
		isIDFieldPresent := false
		for _, field := range td.Fields {
			if field.Key == "id" || field.FromColumn == "id" {
				isIDFieldPresent = true
			}
		}
		// Assume there exists a column in the table called 'id'
		// which is the primary key
		if !isIDFieldPresent {
			field := Field{
				Key:        "id",
				Name:       "Identifier",
				FromColumn: "id",
				Type:       &WorkflowType{Name: "text"},
			}
			td.Fields = append(td.Fields, &field)
		}
	}
	return descriptor
}

func performSanityChecks(descriptor *Descriptor) error {
	for _, td := range descriptor.TypeDescriptors {
		for _, field := range td.Fields {
			if err := errCurrencyHasDefaultValue(field, td.Key); err != nil {
				return err
			}
			if err := errFromColumnPropertyIsMissing(field); err != nil {
				return err
			}
		}
	}
	return nil
}

func errCurrencyHasDefaultValue(field *Field, td string) error {
	msg := "Unable to parse descriptor.json: " +
		"%s.%s specifies a default currency value" +
		"*and* a fromColumn. You must specify *only* one."
	if field.Type.Name == "money" {
		if field.Type.Currency.Value != "" &&
			field.Type.Currency.FromColumn != "" {
			return fmt.Errorf(
				msg,
				td,
				field.Key,
			)
		}
	}
	return nil
}

func errFromColumnPropertyIsMissing(field *Field) error {
	msg := "Unable to parse descriptor.json: " +
		"field of type '%s' should contain a fromColumn property"
	if field.Type.Name != "money" && field.Relationship == nil {
		if field.FromColumn == "" {
			return fmt.Errorf(msg, field.Type.Name)
		}
	}
	return nil
}
