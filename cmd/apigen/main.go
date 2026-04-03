// apigen generates an OpenAPI 3.0 specification from Go structs and route
// definitions using openapi3gen and the apidef route metadata.
package main

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"git.duti.dev/secure-package-registry/pkg/apidef"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/external"
	"git.duti.dev/secure-package-registry/pkg/services/core-svc/handlers/private"
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

	// Collect all route definitions.
	allRoutes := append(external.Routes(), private.Routes()...)

	// Collect all request/response types referenced by routes.
	var routeTypes []any
	seen := make(map[reflect.Type]bool)
	for _, route := range allRoutes {
		for _, t := range []reflect.Type{route.RequestBody, route.ResponseType} {
			if t != nil && !seen[t] {
				seen[t] = true
				routeTypes = append(routeTypes, reflect.New(t).Interface())
			}
		}
	}

	// Generate schemas from the pkgdb model types (may overlap with route types).
	if err := generateSchemas(schemas, enumTypes,
		&pkgdb.PackageVersion{},
		&pkgdb.TagError{},
		&pkgdb.PackageSummary{},
		&pkgdb.SearchResult{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating schemas: %v\n", err)
		os.Exit(1)
	}

	// Generate schemas for all route request/response types.
	if err := generateSchemas(schemas, enumTypes, routeTypes...); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating route type schemas: %v\n", err)
		os.Exit(1)
	}

	// Add enum schemas for types discovered with enum tags.
	addEnumSchemas(schemas, enumTypes)

	// Replace inline enums with $ref to named enum schemas.
	deduplicateEnumRefs(schemas)

	// Also generate an ErrorResponse schema.
	errSchema := openapi3.NewObjectSchema()
	errSchema.Properties = openapi3.Schemas{
		"error": &openapi3.SchemaRef{
			Value: openapi3.NewStringSchema(),
		},
	}
	errSchema.Required = []string{"error"}
	schemas["ErrorResponse"] = &openapi3.SchemaRef{Value: errSchema}

	// Build paths from route definitions.
	paths := buildPaths(allRoutes, schemas)

	// Create OpenAPI document.
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Secure Package Registry API",
			Description: "API for querying package information and metadata",
			Version:     "1.0.0",
		},
		Paths: paths,
		Components: &openapi3.Components{
			Schemas: schemas,
		},
	}

	// Create output directory if it doesn't exist.
	outputDir := "docs/svc"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Marshal to YAML.
	data, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling OpenAPI doc: %v\n", err)
		os.Exit(1)
	}

	// Write to file.
	outputPath := outputDir + "/openapi.yaml"
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated OpenAPI spec at %s\n", outputPath)
}

// buildPaths constructs the OpenAPI Paths object from route definitions.
func buildPaths(routes []apidef.RouteDef, schemas openapi3.Schemas) *openapi3.Paths {
	paths := openapi3.NewPaths()

	for _, route := range routes {
		// Get or create the path item.
		pathItem := paths.Value(route.Path)
		if pathItem == nil {
			pathItem = &openapi3.PathItem{}
			paths.Set(route.Path, pathItem)
		}

		responses := &openapi3.Responses{}
		op := &openapi3.Operation{
			Summary:     route.Summary,
			Description: route.Description,
			OperationID: route.OperationID,
			Tags:        []string{route.Tag},
			Responses:   responses,
		}

		// Path parameters.
		for _, paramName := range route.PathParams {
			op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:     paramName,
					In:       "path",
					Required: true,
					Schema: &openapi3.SchemaRef{
						Value: openapi3.NewStringSchema(),
					},
				},
			})
		}

		// Query parameters.
		for _, qp := range route.QueryParams {
			op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        qp.Name,
					In:          "query",
					Description: qp.Description,
					Required:    qp.Required,
					Schema: &openapi3.SchemaRef{
						Value: openapi3.NewStringSchema(),
					},
				},
			})
		}

		// Request body.
		if route.RequestBody != nil {
			schemaName := route.RequestBody.Name()
			op.RequestBody = &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/" + schemaName,
							},
						},
					},
				},
			}
		}

		// Success response.
		successCode := fmt.Sprintf("%d", route.SuccessCode)
		successDesc := http.StatusText(route.SuccessCode)

		if route.SuccessCode == 204 {
			// No body.
			op.Responses.Set(successCode, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: &successDesc,
				},
			})
		} else if route.StreamResponse {
			// Raw byte stream.
			ct := route.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			op.Responses.Set(successCode, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: &successDesc,
					Content: openapi3.Content{
						ct: &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"string"},
									Format: "binary",
								},
							},
						},
					},
				},
			})
		} else if route.ResponseType != nil {
			schemaName := route.ResponseType.Name()
			op.Responses.Set(successCode, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: &successDesc,
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Ref: "#/components/schemas/" + schemaName,
							},
						},
					},
				},
			})
		}

		// Error responses — all endpoints use ErrorResponse.
		addErrorResponse(op, "400", "Bad Request")
		if route.Auth {
			addErrorResponse(op, "401", "Unauthorized")
		}
		addErrorResponse(op, "404", "Not Found")
		addErrorResponse(op, "500", "Internal Server Error")

		// Security requirement for authenticated endpoints.
		if route.Auth {
			op.Security = &openapi3.SecurityRequirements{
				{"BearerAuth": {}},
			}
		}

		// Assign the operation to the correct method.
		switch route.Method {
		case "GET":
			pathItem.Get = op
		case "POST":
			pathItem.Post = op
		case "PUT":
			pathItem.Put = op
		case "DELETE":
			pathItem.Delete = op
		case "PATCH":
			pathItem.Patch = op
		}
	}

	return paths
}

// addErrorResponse adds a standard error response to an operation.
func addErrorResponse(op *openapi3.Operation, code, description string) {
	op.Responses.Set(code, &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &description,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/ErrorResponse",
					},
				},
			},
		},
	})
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

// deduplicateEnumRefs replaces inline enum definitions with $ref to named enum schemas
func deduplicateEnumRefs(schemas openapi3.Schemas) {
	// Build a map of enum signature -> schema name
	enumSigs := make(map[string]string)
	for name, schemaRef := range schemas {
		if schemaRef.Value != nil && len(schemaRef.Value.Enum) > 0 {
			sig := buildEnumSig(schemaRef.Value.Enum)
			enumSigs[sig] = name
		}
	}

	// Replace inline enums with $ref in all schema properties
	for _, schemaRef := range schemas {
		if schemaRef.Value == nil || schemaRef.Value.Properties == nil {
			continue
		}
		for _, propRef := range schemaRef.Value.Properties {
			if propRef.Value == nil || len(propRef.Value.Enum) == 0 {
				continue
			}
			sig := buildEnumSig(propRef.Value.Enum)
			if refName, ok := enumSigs[sig]; ok {
				// Replace with $ref to the named enum schema
				propRef.Value.Enum = nil
				propRef.Ref = "#/components/schemas/" + refName
			}
		}
	}
}

// buildEnumSig creates a signature string from enum values for comparison
func buildEnumSig(enum []any) string {
	values := make([]string, len(enum))
	for i, v := range enum {
		values[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(values, ",")
}
