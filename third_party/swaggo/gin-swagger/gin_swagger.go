package ginswagger

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

type Config struct{}

type Option func(*Config)

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      const ui = SwaggerUIBundle({
        url: '{{.DocURL}}',
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "StandaloneLayout"
      });
      window.ui = ui;
    };
  </script>
</body>
</html>`

var tmpl = template.Must(template.New("swagger_index").Parse(indexTemplate))

// WrapHandler returns a Gin handler that serves a minimal Swagger UI backed by the
// documentation registered with github.com/swaggo/swag.
func WrapHandler(handler http.Handler, opts ...Option) gin.HandlerFunc {
	_ = handler
	return func(c *gin.Context) {
		path := c.Param("any")
		switch path {
		case "", "/", "/index.html":
			c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			c.Writer.WriteHeader(http.StatusOK)
			_ = tmpl.Execute(c.Writer, struct{ DocURL string }{DocURL: "./doc.json"})
			return
		case "/doc.json":
			doc := swag.ReadDoc(swag.Name)
			if doc == "" {
				http.NotFound(c.Writer, c.Request)
				return
			}
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(http.StatusOK)
			_, _ = c.Writer.Write([]byte(doc))
			return
		default:
			http.NotFound(c.Writer, c.Request)
		}
	}
}
