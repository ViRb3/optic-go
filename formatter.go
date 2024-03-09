package opticgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"io/ioutil"
)

func Format(specFilePath string) error {
	swagger, err := openapi3.NewLoader().LoadFromFile(specFilePath)
	if err != nil {
		return err
	}
	for _, path := range swagger.Paths.Map() {
		for _, operation := range path.Operations() {
			// Optic sets Summary, while generators expect OperationID. Fix that.
			operation.OperationID = operation.Summary
			for code, response := range operation.Responses.Map() {
				schemaName := fmt.Sprintf("%s_%s_Response", operation.OperationID, code)
				if err := extractSchemas(response.Value.Content, schemaName, swagger); err != nil {
					return err
				}
			}
			schemaName := fmt.Sprintf("%s_Request", operation.OperationID)
			if operation.RequestBody != nil {
				if err := extractSchemas(operation.RequestBody.Value.Content, schemaName, swagger); err != nil {
					return err
				}
			}
		}
	}
	specBytes, err := swagger.MarshalJSON()
	if err != nil {
		return err
	}
	var specBytesIndented bytes.Buffer
	if err := json.Indent(&specBytesIndented, specBytes, "", "  "); err != nil {
		return err
	}
	if err := ioutil.WriteFile(specFilePath, specBytesIndented.Bytes(), 600); err != nil {
		return err
	}
	return nil
}

func extractSchemas(allContent openapi3.Content, schemaName string, swagger *openapi3.T) error {
	i := 2
	for _, content := range allContent {
		// already extracted definition
		if content.Schema.Ref != "" {
			continue
		}
		var targetSchemaRef *openapi3.SchemaRef
		if content.Schema.Value.Items != nil {
			targetSchemaRef = content.Schema.Value.Items
		} else {
			targetSchemaRef = content.Schema
		}
		schemaBytes, err := targetSchemaRef.MarshalJSON()
		if err != nil {
			return err
		}
		if len(allContent) > 1 {
			schemaName += fmt.Sprint(i)
			i++
		}
		schema := openapi3.Schema{}
		if err := schema.UnmarshalJSON(schemaBytes); err != nil {
			return err
		}
		swagger.Components.Schemas[schemaName] = schema.NewRef()
		targetSchemaRef.Ref = "#/components/schemas/" + schemaName
	}
	return nil
}
