package swagger

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fekuna/omnipos-pkg/logger"
	"go.uber.org/zap"
)

//go:embed all:specs
var embeddedSpecsFS embed.FS

// Handler serves OpenAPI specs and Swagger UI
type Handler struct {
	logger     logger.ZapLogger
	isDev      bool
	protoPath  string
	swaggerURL string
}

// NewHandler creates a new Swagger handler
func NewHandler(log logger.ZapLogger) *Handler {
	h := &Handler{
		logger: log,
	}

	protoPath := "../omnipos-proto/openapi"
	absProtoPath, err := filepath.Abs(protoPath)
	if err == nil {
		if _, err := os.Stat(absProtoPath); err == nil {
			h.isDev = true
			h.protoPath = absProtoPath
			h.swaggerURL = "/openapi/user/v1/user.swagger.json"
			log.Info("🚀 Swagger: Development mode",
				zap.String("proto_path", absProtoPath),
				zap.String("mode", "live reload from proto directory"))
		}
	}

	if !h.isDev {
		h.swaggerURL = "/openapi/user.swagger.json"
		log.Info("📦 Swagger: Production mode",
			zap.String("mode", "embedded specs"))
	}

	return h
}

// RegisterRoutes registers Swagger UI routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	if h.isDev {
		// Development: serve directly from proto directory for instant updates
		mux.Handle("/openapi/", http.StripPrefix("/openapi/", http.FileServer(http.Dir(h.protoPath))))
		h.logger.Info("✅ Swagger specs: serving from local proto directory",
			zap.String("path", h.protoPath))
	} else {
		// Production: serve from embedded filesystem
		specsSubFS, err := fs.Sub(embeddedSpecsFS, "specs")
		if err != nil {
			h.logger.Fatal("failed to create specs sub filesystem", zap.Error(err))
		}
		mux.Handle("/openapi/", http.StripPrefix("/openapi/", http.FileServer(http.FS(specsSubFS))))
		h.logger.Info("✅ Swagger specs: serving from embedded filesystem")
	}

	// Serve Swagger UI
	mux.HandleFunc("/swagger-ui", h.serveSwaggerUI)
	mux.HandleFunc("/swagger-ui/", h.serveSwaggerUI)

	h.logger.Info("📖 Swagger UI available",
		zap.String("url", "http://localhost:8081/swagger-ui"),
		zap.Bool("dev_mode", h.isDev))
}

// serveSwaggerUI serves a standalone Swagger UI HTML page
func (h *Handler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// Get the server URL from the request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	serverURL := scheme + "://" + r.Host

	// Define available specs
	// Note: URLs must match the file structure served by the file server
	// /openapi/ maps to the root of the embedded specs or local proto directory
	specUrls := []struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}{
		{URL: "/openapi/user/v1/user.swagger.json", Name: "Merchant API"},
		{URL: "/openapi/product/v1/product.swagger.json", Name: "Product API"},
		{URL: "/openapi/product/v1/inventory.swagger.json", Name: "Inventory API"},
	}

	// Generate the urls array for Swagger UI
	// We construct the JS array string manually to avoid complex templating dependencies
	urlsJS := "[\n"
	for _, spec := range specUrls {
		urlsJS += "                    {url: '" + spec.URL + "', name: '" + spec.Name + "'},\n"
	}
	urlsJS += "                ]"

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OmniPOS API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
        .swagger-ui .topbar {
            background-color: #1a1a1a;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                urls: ` + urlsJS + `,
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                persistAuthorization: true,
                filter: true,
                tryItOutEnabled: true,
                displayRequestDuration: true,
                // Force server URL to current host
                servers: [
                    { url: '` + serverURL + `', description: 'Gateway Server' }
                ],
                // Intercept requests to force HTTP
                requestInterceptor: (req) => {
                    // Force current server's protocol
                    if (req.url) {
                        req.url = req.url.replace('https://', 'http://');
                    }
                    return req;
                }
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
