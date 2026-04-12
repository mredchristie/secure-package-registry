package private

import (
	"reflect"

	"git.duti.dev/secure-package-registry/pkg/apidef"
)

// Routes returns the route definitions for all internal API endpoints.
// These are consumed by cmd/apigen to generate the OpenAPI paths section.
func Routes() []apidef.RouteDef {
	return []apidef.RouteDef{
		{
			Method:       "POST",
			Path:         "/api/v1/internal/packages",
			Summary:      "Create package (internal)",
			Description:  "Register a package from an internal service. Returns existing package if already created.",
			Tag:          "Internal",
			OperationID:  "internalCreatePackage",
			RequestBody:  reflect.TypeFor[CreatePackageRequest](),
			ResponseType: reflect.TypeFor[CreatePackageResponse](),
			SuccessCode:  201,
		},
		{
			Method:       "POST",
			Path:         "/api/v1/internal/reproducible-builds",
			Summary:      "Register reproducible build (internal)",
			Description:  "Validate that a package version exists, tag it as reproducible, and upload the artifact to the registry.",
			Tag:          "Internal",
			OperationID:  "internalRegisterReproducibleBuild",
			RequestBody:  reflect.TypeFor[RegisterReproducibleBuildRequest](),
			ResponseType: reflect.TypeFor[RegisterReproducibleBuildResponse](),
			SuccessCode:  201,
		},
	}
}
