#!/usr/bin/env python

import json
from jinja2 import Template
from optparse import OptionParser

type_descriptor_template = """{
      "key" : "{{ key }}",
      "name" : "{{ name }}",
      "tableName": "{{ tableName }}",
      "columnAsOptionName": "{{ columnAsOptionName }}",
      "uniqueIdColumn": "{{ uniqueIdColumn }}",
      "recordType": "{{ recordType }}",
      "fields" : [
      {% for field in fields -%}
      {% if field.type == "money" %}
        {
          "key" : "{{ field.key }}",
          "name" : "{{ field.name }}",
          "type" : {
            "name" : "money",
             "amount": {
               "key": "{{ field.key }}",
               "fromColumn": "{{ field.fromColumn }}"
             },
             "currency": {
               "value": "EUR"
             }
          }
        }{% endif %}{%- if field.type == "text" %}
        {
          "key": "{{ field.key }}",
          "name": "{{ field.name }}",
          "fromColumn": "{{ field.fromColumn }}",
          "type": {
            "name": "text"
          }
        }{% endif %}{%- if field.type == "date" %}
        {
          "key" : "{{ field.key }}",
          "name" : "{{ field.name }}",
          "fromColumn" : "{{ field.fromColumn }}",
          "type" : {
            "name" : "date",
            "kind": "date"
          }
        }{% endif %}{%- if field.type == "number" %}
        {
          "key" : "{{ field.key }}",
          "name" : "{{ field.name }}",
          "fromColumn" : "{{ field.fromColumn }}",
          "type" : {
            "name" : "number"
          }
        }
        {% endif %}{%- if field.type == "datetime" %}
        {
          "key" : "{{ field.key }}",
          "name" : "{{ field.name }}",
          "fromColumn" : "{{ field.fromColumn }}",
          "type" : {
            "name" : "date",
            "kind": "datetime"
          }
        }{% endif %}{%- if loop.last == False %},{% endif %}
      {% endfor %}
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
}"""

def main():
    usage = """
    %prog [options] <table_schema_file>

    The table_schema_file is a pipe delimited table which
    contains the table column name in the first column
    and the associated workflow data type in the
    second column, for example:

    | emailAddress    | text     |
    | creationDate    | dateTime |
    | acquisitionCost | money    |
    """
    parser = OptionParser(usage)
    parser.add_option("-k", "--key", dest="key",
                    help="Set the `key` for the type descriptor")
    parser.add_option("-t", "--table-name", dest="table_name",
                    help="Set the `tableName` for the type descriptor")
    parser.add_option("-c", "--column-as-option-name", dest="column_as_option_name",
                    help="Set the `columnAsOptionName` for the type descriptor")
    parser.add_option("-u", "--unique-id-column", dest="unique_id_column",
                    help="Set the `uniqueIdColumn` for the type descriptor")
    (options, args) = parser.parse_args()
    if len(args) != 1:
        parser.error("You must provide the table schema file as argument")
    template = Template(type_descriptor_template)
    fields = populateFieldsFrom(args[0])
    print(generateDescriptor(options, template, fields))

def camelCase(s):
    output = ''.join(x for x in s.title() if x.isalnum())
    return output[0].lower() + output[1:]

def populateFieldsFrom(file):
    fields = []
    with open(file) as f:
        for line in f:
            fromColumn, fieldType = line.split('|')[1:3]
            fields.append(
                {
                    'fromColumn': fromColumn.strip(),
                    'type': fieldType.strip(),
                    'key': camelCase(fromColumn.strip()),
                    'name': fromColumn.strip()
                }
            )
    return fields

def generateDescriptor(options, template, fields):
    result = json.dumps(
        json.loads(
            (template.render(
                key=options.key,
                name=camelCase(options.key),
                tableName=options.table_name,
                columnAsOptionName=options.column_as_option_name,
                uniqueIdColumn=options.unique_id_column,
                recordType="value",
                fields=fields))),
        indent=2,
        ensure_ascii=False
    )
    return result

if __name__ == "__main__":
    main()
