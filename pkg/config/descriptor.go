package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

type Descriptor struct {
	Key             string
	Name            string
	Description     string
	TypeDescriptors []*TypeDescriptor
}

type TypeDescriptor struct {
	Key                string
	Name               string
	TableName          string
	ColumnAsOptionName string
	Fields             []*Field
	OptionsAvailable   bool
	FetchOneAvailable  bool
}

type Field struct {
	Key          string
	Name         string
	Type         *WorkflowType
	FromColumn   string
	Relationship *Relationship
	Amount       struct {
		Key        string
		FromColumn string
	}
	Currency struct {
		Key        string
		FromColumn string
		Value      string
	}
}

type WorkflowType struct {
	Name string
	Kind string
}

type Relationship struct {
	WithTable  string
	ForeignKey string
}

// ParseDescriptorFile will parse the descriptor.json file and make sure
// to add an `id` field if the user has not already specified it
func ParseDescriptorFile(file io.Reader) (descriptor *Descriptor) {
	var content []byte
	content, err := ioutil.ReadAll(file)
	if err != nil {
		panic(fmt.Errorf("Unable to parse descriptor.json file: %v", err))
	}
	err = json.Unmarshal(content, &descriptor)
	if err != nil {
		panic(fmt.Errorf("Unable to unmarshal descriptor.json: %v", err))
	}
	descriptorWithID := addIDFieldIfNotExists(descriptor)
	return descriptorWithID
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
