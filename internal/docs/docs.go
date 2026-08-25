// Package docs serves the OpenAPI specification and Scalar API reference UI.
package docs

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var openAPISpec []byte

const scalarPage = `<!doctype html>
<html lang="en">
  <head>
    <title>futureEnvirons API</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference/dist/browser/standalone.js"></script>
    <script>
      Scalar.createApiReference('#app', {
        url: '/docs/openapi.yaml',
        theme: 'default',
      })
    </script>
  </body>
</html>
`

// Register mounts Scalar UI and the embedded OpenAPI document on the Gin engine.
func Register(r *gin.Engine) {
	r.GET("/docs", scalarUI)
	r.GET("/docs/openapi.yaml", openAPISpecHandler)
}

func scalarUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, scalarPage)
}

func openAPISpecHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml", openAPISpec)
}
