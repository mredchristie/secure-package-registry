// apigen generates OpenAPI 3.0 specification from Go structs using openapi3gen.
package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"gopkg.in/yaml.v3"
)

func main() {
	// Create schemas map
	schemas := make(openapi3.Schemas)

	// Map to collect enum types found during schema generation
	// key: type name (e.g., "PkgVtype"), value: []enum values
	enumTypes := make(map[string][]string)

	// Generate schemas from struct instances (pass pointers)
	if err := generateSchemas(schemas, enumTypes,
		&pkgdb.PackageVersion{},
		&pkgdb.TagError{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating schemas: %v\n", err)
		os.Exit(1)
	}

	// Add enum schemas for types discovered with enum tags
	addEnumSchemas(schemas, enumTypes)

	// Create OpenAPI document
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Secure Package Registry API",
			Description: "API for querying package information and metadata",
			Version:     "1.0.0",
		},
		Components: &openapi3.Components{
			Schemas: schemas,
		},
	}

	// Create output directory if it doesn't exist
	outputDir := "docs/svc"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling OpenAPI doc: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	outputPath := outputDir + "/openapi.yaml"
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated OpenAPI spec at %s\n", outputPath)
}

// generateSchemas generates OpenAPI schemas for multiple struct types.
// Pass pointers to structs, e.g.: generateSchemas(schemas, &pkgdb.PackageVersion{}, &pkgdb.TagError{})
func generateSchemas(schemas openapi3.Schemas, enumTypes map[string][]string, instances ...any) error {
	// Custom schema customizer that extracts enum values from struct tags
	customizer := openapi3gen.SchemaCustomizer(func(name string, ft reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		// Check for enum tag
		if enumTag := tag.Get("enum"); enumTag != "" {
			// Parse comma-separated enum values
			values := strings.Split(enumTag, ",")
			for i := range values {
				values[i] = strings.TrimSpace(values[i])
			}

			// Get the type name
			typeName := getTypeName(ft)
			if typeName != "" {
				// Store enum values for this type
				enumTypes[typeName] = values
			}

			// Also add enum to the field schema
			for _, v := range values {
				schema.Enum = append(schema.Enum, v)
			}
		}
		return nil
	})

	opts := []openapi3gen.Option{
		openapi3gen.CreateComponentSchemas(openapi3gen.ExportComponentSchemasOptions{
			ExportComponentSchemas: true,
			ExportTopLevelSchema:   true,
		}),
		openapi3gen.UseAllExportedFields(),
		customizer,
	}

	for _, instance := range instances {
		// instance is already a pointer (*pkgdb.PackageVersion, etc.)
		// no need for & since it's already a pointer
		if _, err := openapi3gen.NewSchemaRefForValue(instance, schemas, opts...); err != nil {
			return fmt.Errorf("error generating schema: %w", err)
		}
	}

	return nil
}

// getTypeName extracts a clean type name from reflect.Type
func getTypeName(t reflect.Type) string {
	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Get the type name
	name := t.Name()
	if name == "" {
		return ""
	}

	// If it's from a package, return just the name
	// (e.g., "coredb.PkgVtype" -> "PkgVtype")
	if t.PkgPath() != "" {
		return name
	}

	return name
}

// addEnumSchemas creates enum schemas from discovered types
func addEnumSchemas(schemas openapi3.Schemas, enumTypes map[string][]string) {
	for typeName, values := range enumTypes {
		// Convert values to []any for OpenAPI
		enumValues := make([]any, len(values))
		for i, v := range values {
			enumValues[i] = v
		}

		// Create enum schema
		schemas[typeName] = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
				Enum: enumValues,
			},
		}
	}
}
