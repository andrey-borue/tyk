package swagger

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"github.com/TykTechnologies/tyk/apidef/oas"
	"github.com/TykTechnologies/tyk/gateway"
)

const OASTag = "OAS APIs"

func OasAPIS(r *openapi3.Reflector) error {
	return addOperations(r, getListOfOASApisRequest, postOAsApi, apiOASExportHandler, getOASApiRequest, apiOASPutHandler, deleteOASHandler, apiOASExportWithIDHandler, importApiOASPostHandler, oasVersionsHandler, apiOASPatchHandler)
}

var responseSchema = jsonschema.Schema{
	Ref: PointerValue("https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/schemas/v3.0/schema.json"),
}
var responseSchemaWithExtension = jsonschema.AllOf(responseSchema, oas.XTykAPIGateway{})

func getListOfOASApisRequest(r *openapi3.Reflector) error {
	op, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodGet,
		PathPattern: "/tyk/apis/oas",
		OperationID: "listApisOAS",
		Tag:         OASTag,
	})
	oc := op.oc
	if err != nil {
		return err
	}
	//TODO
	///recommended i use external reference just incase it is updated
	op.AddQueryParameter("mode", "Mode of OAS get, by default mode could be empty which means to get OAS spec including OAS Tyk extension. \n When mode=public, OAS spec excluding Tyk extension will be returned in the response", OptionalParameterValues{
		Example: valueToInterface("public"),
		Enum:    []interface{}{"public"},
	})

	oc.SetSummary("List all OAS format APIS")
	oc.SetDescription("List all OAS format APIs, when used without the Tyk Dashboard.")
	item := []jsonschema.AllOfExposer{jsonschema.AllOf(responseSchema, oas.XTykAPIGateway{})}
	op.AddRespWithRefExamples(http.StatusOK, item, []multipleExamplesValues{
		{
			key:         oasExampleList,
			httpStatus:  200,
			Summary:     "List of API definitions in OAS format",
			exampleType: Component,
			ref:         oasExampleList,
			hasExample:  true,
		},
	}, func(cu *openapi.ContentUnit) {
		cu.Description = "List of API definitions in OAS format"
	})
	return op.AddOperation()
}

func postOAsApi(r *openapi3.Reflector) error {
	// TODO::Should this be external reference or should we create a local object.
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodPost,
		PathPattern: "/tyk/apis/oas",
		OperationID: "createApiOAS",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	oc.StatusBadRequest("the payload should contain x-tyk-api-gateway")
	oc.StatusInternalServerError("file object creation failed, write error")
	oc.AddRespWithExample(apiModifyKeySuccess{
		Key:    "e30bee13ad4248c3b529a4c58bb7be4e",
		Status: "ok",
		Action: "added",
	}, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API created"
	})
	oc.SetDescription("Create API with OAS format\n         A single Tyk node can have its API Definitions queried, deleted and updated remotely. This functionality enables you to remotely update your Tyk definitions without having to manage the files manually.")
	oc.SetSummary("Create API with OAS format")
	oc.AddReqWithSeparateExample(responseSchemaWithExtension, oasSample(OasSampleString()))
	addApiPostQueryParam(oc)
	///addExternalRefToRequest(o3)
	return oc.AddOperation()
}

func apiOASExportHandler(r *openapi3.Reflector) error {
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodGet,
		PathPattern: "/tyk/apis/oas/export",
		OperationID: "downloadApisOASPublic",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	oc.SetSummary("Download all OAS format APIs")
	oc.SetDescription("Download all OAS format APIs, when used without the Tyk Dashboard.")
	oc.AddBinaryFormatResp(BinaryFormat{
		///example:     BinaryExample(OasSampleString()),
		httpStatus:  200,
		description: "Get list of oas API definition",
	})
	oc.StatusInternalServerError("Unexpected error")
	oc.AddQueryParameter("mode", "Mode of OAS get, by default mode could be empty which means to get OAS spec including OAS Tyk extension. \n When mode=public, OAS spec excluding Tyk extension will be returned in the response", OptionalParameterValues{
		Required: PointerValue(false),
		Example:  valueToInterface("public"),
		Type:     openapi3.SchemaTypeString,
		Enum:     []interface{}{"public"},
	})

	return oc.AddOperation()
}

// Done
func getOASApiRequest(r *openapi3.Reflector) error {
	// TODO::response of this is different from previous
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodGet,
		PathPattern: "/tyk/apis/oas/{apiID}",
		OperationID: "getOASApi",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	oc.StatusNotFound("API not found", func(cu *openapi.ContentUnit) {
		cu.Description = "API not found"
	})
	oc.StatusBadRequest("the requested API definition is in Tyk classic format, please use old api endpoint")
	oc.SetSummary("Get OAS Api definition")
	oc.SetDescription("Get OAS Api definition\n  using the api Id")
	oc.AddResponseHeaders(ResponseHeader{
		Name:        "x-tyk-base-api-id",
		Description: PointerValue("ID of the base API if the requested API is a version."),
		Type:        PointerValue(openapi3.SchemaTypeString),
	})
	oc.AddQueryParameter("mode", "Mode of OAS get, by default mode could be empty which means to get OAS spec including OAS Tyk extension. \n When mode=public, OAS spec excluding Tyk extension will be returned in the response", OptionalParameterValues{
		Required: PointerValue(false),
		Example:  valueToInterface("public"),
		Type:     openapi3.SchemaTypeString,
		Enum:     []interface{}{"public"},
	})
	oc.AddPathParameter("apiID", "ID of the api you want to fetch", OptionalParameterValues{
		Example: valueToInterface("4c1c0d8fc885401053ddac4e39ef676b"),
	})

	oc.AddRespWithRefExamples(http.StatusOK, responseSchemaWithExtension, []multipleExamplesValues{
		{
			object:      nil,
			key:         oasExample,
			httpStatus:  200,
			Summary:     "Api fetched successfully",
			exampleType: Component,
			ref:         oasExample,
		},
	})
	return oc.AddOperation()
}

// Done
func apiOASPutHandler(r *openapi3.Reflector) error {
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodPut,
		PathPattern: "/tyk/apis/oas/{apiID}",
		OperationID: "updateApiOAS",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	oc.StatusInternalServerError("file object creation failed, write error")
	oc.StatusBadRequest("Request APIID does not match that in Definition! For Update operations these must match.")
	oc.StatusNotFound("API not found", func(cu *openapi.ContentUnit) {
		cu.Description = "API not found"
	})
	oc.AddRespWithExample(apiModifyKeySuccess{
		Key:    "e30bee13ad4248c3b529a4c58bb7be4e",
		Status: "ok",
		Action: "modified",
	}, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API updated"
	})
	oc.SetSummary("Update OAS API definition")
	oc.SetDescription("Updating an API definition uses the same signature an object as a `POST`, however it will first ensure that the API ID that is being updated is the same as the one in the object being `PUT`.\n\n\n        Updating will completely replace the file descriptor and will not change an API Definition that has already been loaded, the hot-reload endpoint will need to be called to push the new definition to live.")
	oc.AddReqWithSeparateExample(responseSchemaWithExtension, oasSample(OasSampleString()))
	oc.AddPathParameter("apiID", "ID of the api you want to fetch", OptionalParameterValues{
		Example: valueToInterface("4c1c0d8fc885401053ddac4e39ef676b"),
	})
	return oc.AddOperation()
}

func apiOASExportWithIDHandler(r *openapi3.Reflector) error {
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodGet,
		PathPattern: "/tyk/apis/oas/{apiID}/export",
		OperationID: "downloadApiOASPublic",
		Tag:         OASTag,
	})
	///TODO:: should we add Content-Disposition headers
	if err != nil {
		return err
	}
	oc.StatusInternalServerError("Unexpected error")
	oc.StatusBadRequest("requesting API definition that is in Tyk classic format")
	oc.StatusNotFound("API not found")
	oc.AddPathParameter("apiID", "ID of the api you want to fetch", OptionalParameterValues{
		Example: valueToInterface("4c1c0d8fc885401053ddac4e39ef676b"),
	})
	oc.AddQueryParameter("mode", "Mode of OAS get, by default mode could be empty which means to get OAS spec including OAS Tyk extension. \n When mode=public, OAS spec excluding Tyk extension will be returned in the response", OptionalParameterValues{
		Required: PointerValue(false),
		Example:  valueToInterface("public"),
		Type:     openapi3.SchemaTypeString,
		Enum:     []interface{}{"public"},
	})
	oc.AddBinaryFormatResp(BinaryFormat{
		///example:     BinaryExample(OasSampleString()),
		httpStatus:  200,
		description: "Exported API definition file",
	})
	oc.SetSummary("Download an OAS format APIs, when used without the Tyk Dashboard.")
	oc.SetDescription("Mode of OAS export, by default mode could be empty which means to export OAS spec including OAS Tyk extension. \n  When mode=public, OAS spec excluding Tyk extension is exported")

	return oc.AddOperation()
}

func importApiOASPostHandler(r *openapi3.Reflector) error {
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodPost,
		PathPattern: "/tyk/apis/oas/import",
		OperationID: "importOAS",
		Tag:         OASTag,
	})
	///TODO:: check if the OAs post query parameters can be applied here.
	if err != nil {
		return err
	}
	oc.SetSummary("Create a new OAS format API, without x-tyk-gateway")
	oc.AddRespWithExample(apiModifyKeySuccess{
		Key:    "e30bee13ad4248c3b529a4c58bb7be4e",
		Status: "ok",
		Action: "added",
	}, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API imported"
	})
	oc.SetDescription("Create a new OAS format API, without x-tyk-gateway.\n        For use with an existing OAS API that you want to expose via your Tyk Gateway. (New)")
	oc.StatusInternalServerError("file object creation failed, write error")
	oc.StatusBadRequest("the import payload should not contain x-tyk-api-gateway")
	importAndPatchQueryParameters(oc)
	addApiPostQueryParam(oc)
	oc.AddReqWithSeparateExample(responseSchema, oasSample(OasNoXTykSample()))
	return oc.AddOperation()
}

// Done
func oasVersionsHandler(r *openapi3.Reflector) error {
	// TODO::in previous api this was wrong
	op, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodGet,
		PathPattern: "/tyk/apis/oas/{apiID}/versions",
		OperationID: "listOASApiVersions",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	op.AddPathParameter("apiID", "ID of the api you want to fetch", OptionalParameterValues{
		Example: valueToInterface("4c1c0d8fc885401053ddac4e39ef676b"),
	})
	oc := op.oc
	op.StatusNotFound("API not found", func(cu *openapi.ContentUnit) {
		cu.Description = "API not found"
	})
	oc.SetDescription("Listing versions of an OAS API")

	versionMetas := gateway.VersionMetas{
		Status: "success",
		Metas: []gateway.VersionMeta{
			{
				ID:               "keyless",
				Name:             "Tyk Test Keyless API",
				VersionName:      "",
				Internal:         false,
				ExpirationDate:   "",
				IsDefaultVersion: false,
			},
			{
				ID:               "1f20d5d2731d47ac9c79fddf826eda00",
				Name:             "Version three Api",
				VersionName:      "v2",
				Internal:         false,
				ExpirationDate:   "",
				IsDefaultVersion: true,
			},
		},
	}

	op.AddRespWithExample(versionMetas, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API version metas"
	})
	oc.SetSummary("Listing versions of an OAS API")
	op.AddRefParameters(SearchText)
	op.AddRefParameters(AccessType)

	return op.AddOperation()
}

// /Done
func deleteOASHandler(r *openapi3.Reflector) error {
	op, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodDelete,
		PathPattern: "/tyk/apis/oas/{apiID}",
		OperationID: "deleteOASApi",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	oc := op.oc
	op.StatusInternalServerError("Delete failed")
	op.StatusBadRequest("Must specify an apiID to delete")
	op.StatusNotFound("API not found", func(cu *openapi.ContentUnit) {
		cu.Description = "API not found"
	})
	op.AddRespWithExample(apiModifyKeySuccess{
		Key:    "1bd5c61b0e694082902cf15ddcc9e6a7",
		Status: "ok",
		Action: "deleted",
	}, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API deleted"
	})
	oc.SetSummary("Deleting an OAS API")
	oc.SetDescription("Deleting an API definition will remove the file from the file store, the API definition will NOT be unloaded, a separate reload request will need to be made to disable the API endpoint.")
	op.AddPathParameter("apiID", "The API ID", OptionalParameterValues{
		Example: valueToInterface("1bd5c61b0e694082902cf15ddcc9e6a7"),
	})
	return op.AddOperation()
}

func apiOASPatchHandler(r *openapi3.Reflector) error {
	// TODO;//check this quesry parameters
	oc, err := NewOperationWithSafeExample(r, SafeOperation{
		Method:      http.MethodPatch,
		PathPattern: "/tyk/apis/oas/{apiID}",
		OperationID: "patchApiOAS",
		Tag:         OASTag,
	})
	if err != nil {
		return err
	}
	oc.StatusInternalServerError("file object creation failed, write error")
	oc.StatusBadRequest("Must specify an apiID to patch")
	oc.StatusNotFound("API not found", func(cu *openapi.ContentUnit) {
		cu.Description = "API not found"
	})
	oc.AddResp(apiModifyKeySuccess{
		Key:    "4c1c0d8fc885401053ddac4e39ef676b",
		Status: "ok",
		Action: "modified",
	}, http.StatusOK, func(cu *openapi.ContentUnit) {
		cu.Description = "API patched"
	})
	oc.SetSummary("Patch API with OAS format.")
	oc.SetDescription("Update API with OAS format. You can use this endpoint to update OAS part of the tyk API definition.\n        This endpoint allows you to configure tyk OAS extension based on query params provided(similar to import)")
	oc.AddReqWithSeparateExample(responseSchema, oasSample(OasSampleString()))

	oc.AddPathParameter("apiID", "ID of the api you want to fetch", OptionalParameterValues{
		Example: valueToInterface("4c1c0d8fc885401053ddac4e39ef676b"),
	})
	importAndPatchQueryParameters(oc)
	return oc.AddOperation()
}

type BinarySchema struct {
	Name string `json:"name"`
}

func importAndPatchQueryParameters(oc *OperationWithExample) {
	oc.AddRefParameters(UpstreamURL)
	oc.AddRefParameters(ListenPath)
	oc.AddRefParameters(CustomDomain)
	oc.AddRefParameters(AllowList)
	oc.AddRefParameters(ValidateRequest)
	oc.AddRefParameters(MockResponse)
	oc.AddRefParameters(Authentication)
}

func OasSampleString() string {
	jsonData := `{
		  "openapi": "3.0.3",
		   "info": {
			"title": "OAS Sample",
			"description": "This is a sample OAS.",
			"version": "1.0.0"
		  },
		  "servers": [
			{
			  "url": "https://localhost:8080"
			}
		  ],
		  "security": [
			{
			  "bearerAuth": []
			}
		  ],
		  "paths": {
			"/api/sample/users": {
			  "get": {
				"tags": [
				  "users"
				],
				"summary": "Get users",
				"operationId": "getUsers",
				"responses": {
				  "200": {
					"description": "fetched users",
					"content": {
					  "application/json": {
						"schema": {
						  "type": "array",
						  "items": {
							"type": "object",
							"properties": {
							  "name": {
								"type": "string"
							  }
							}
						  }
						}
					  }
					}
				  }
				}
			  }
			}
		  },
		   "components": {
			"securitySchemes": {
			  "bearerAuth": {
				"type": "http",
				"scheme": "bearer",
				"description": "The API Access Credentials"
			  }
			}
		  },
			"x-tyk-api-gateway": {
					"info": {
						"name": "user",
						"state": {
							"active": true
						}
					},
					"upstream": {
						"url": "https://localhost:8080"
					},
					"server": {
						"listenPath": {
							"value": "/user-test/",
							"strip": true
						}
					}
				}
    }`
	return jsonData
}

func oasSample(data string) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}
	return result
}

func OasNoXTykSample() string {
	jsonData := `
{
  "openapi": "3.0.3",
  "info": {
    "title": "OAS Sample",
    "description": "This is a sample OAS.",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "https://localhost:8080"
    }
  ],
  "security": [
    {
      "bearerAuth": []
    }
  ],
  "paths": {
    "/api/sample/users": {
      "get": {
        "tags": [
          "users"
        ],
        "summary": "Get users",
        "operationId": "getUsers",
        "responses": {
          "200": {
            "description": "fetched users",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "name": {
                        "type": "string"
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "description": "The API Access Credentials"
      }
    }
  }
}`
	return jsonData
}