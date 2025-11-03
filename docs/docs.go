package docs

import "github.com/swaggo/swag"

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "description": "REST API for the Cortex Network Scanner.",
    "title": "Cortex API",
    "termsOfService": "http://swagger.io/terms/",
    "contact": {
      "email": "support@swagger.io",
      "name": "API Support",
      "url": "http://www.swagger.io/support"
    },
    "license": {
      "name": "MIT",
      "url": "https://opensource.org/licenses/MIT"
    },
    "version": "5.0"
  },
  "host": "localhost:8080",
  "basePath": "/api/v1",
  "schemes": [
    "http"
  ],
  "paths": {
    "/scans": {
      "post": {
        "consumes": [
          "application/json"
        ],
        "produces": [
          "application/json"
        ],
        "summary": "Create a new scan task",
        "description": "Accepts a scan request, queues it for processing, and returns a task ID.",
        "operationId": "createScan",
        "tags": [
          "Scans"
        ],
        "security": [
          {
            "ApiKeyAuth": []
          }
        ],
        "parameters": [
          {
            "description": "Scan Request Parameters",
            "name": "scanRequest",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/CreateScanRequest"
            }
          }
        ],
        "responses": {
          "202": {
            "description": "Scan task accepted",
            "schema": {
              "$ref": "#/definitions/AcceptedResponse"
            }
          },
          "400": {
            "description": "Invalid request payload",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "401": {
            "description": "Unauthorized",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "429": {
            "description": "Rate limit exceeded",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "500": {
            "description": "Internal server error",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    },
    "/scans/{id}": {
      "get": {
        "produces": [
          "application/json"
        ],
        "summary": "Get scan status and results",
        "description": "Retrieves the complete details of a scan task by its ID.",
        "operationId": "getScan",
        "tags": [
          "Scans"
        ],
        "security": [
          {
            "ApiKeyAuth": []
          }
        ],
        "parameters": [
          {
            "type": "string",
            "description": "Scan Task ID (UUID)",
            "name": "id",
            "in": "path",
            "required": true
          }
        ],
        "responses": {
          "200": {
            "description": "Full scan task object with results",
            "schema": {
              "$ref": "#/definitions/ScanTask"
            }
          },
          "404": {
            "description": "Task not found",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "401": {
            "description": "Unauthorized",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "429": {
            "description": "Rate limit exceeded",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          },
          "500": {
            "description": "Internal server error",
            "schema": {
              "$ref": "#/definitions/ErrorResponse"
            }
          }
        }
      }
    }
  },
  "securityDefinitions": {
    "ApiKeyAuth": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header"
    }
  },
  "definitions": {
    "AcceptedResponse": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "example": "a3f5c62e-1234-4f72-a84a-1c2d3e4f5678"
        },
        "status": {
          "type": "string",
          "example": "pending"
        }
      },
      "additionalProperties": false
    },
    "CreateScanRequest": {
      "type": "object",
      "required": [
        "hosts",
        "mode",
        "ports"
      ],
      "properties": {
        "hosts": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "example": [
            "scanme.nmap.org",
            "127.0.0.1"
          ]
        },
        "mode": {
          "type": "string",
          "enum": [
            "connect",
            "syn",
            "udp"
          ],
          "example": "connect"
        },
        "ports": {
          "type": "string",
          "example": "22-80"
        }
      },
      "additionalProperties": false
    },
    "ErrorResponse": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string",
          "example": "failed to queue task"
        }
      },
      "additionalProperties": false
    },
    "ScanResult": {
      "type": "object",
      "properties": {
        "host": {
          "type": "string",
          "example": "scanme.nmap.org"
        },
        "port": {
          "type": "integer",
          "format": "int32",
          "example": 80
        },
        "service": {
          "type": "string",
          "example": "http",
          "x-nullable": true
        },
        "state": {
          "type": "string",
          "example": "open"
        }
      },
      "additionalProperties": false
    },
    "ScanTask": {
      "type": "object",
      "properties": {
        "completed_at": {
          "type": "string",
          "format": "date-time"
        },
        "created_at": {
          "type": "string",
          "format": "date-time",
          "example": "2024-01-02T15:04:05Z"
        },
        "error": {
          "type": "string",
          "example": "failed to queue task"
        },
        "hosts": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "id": {
          "type": "string",
          "example": "a3f5c62e-1234-4f72-a84a-1c2d3e4f5678"
        },
        "mode": {
          "type": "string",
          "example": "connect"
        },
        "ports": {
          "type": "string",
          "example": "22-80"
        },
        "results": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/ScanResult"
          }
        },
        "status": {
          "type": "string",
          "example": "pending"
        }
      },
      "additionalProperties": false
    }
  }
}
`

func init() {
	swag.Register(swag.Name, &swaggerDoc{})
}

type swaggerDoc struct{}

func (s *swaggerDoc) ReadDoc() string {
	return docTemplate
}
