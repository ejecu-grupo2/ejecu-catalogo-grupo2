package api

import (
	"context"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"net/http"
)

type Disponibilidad interface{ Disponible(context.Context) error }
type Salida struct{ Body service.Estado }

func Registrar(mux *http.ServeMux, entorno string, db Disponibilidad) huma.API {
	config := huma.DefaultConfig("Catálogo", "0.1.0")
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	config.SchemasPath = "/api/schemas"
	if entorno == "prod" {
		config.OpenAPIPath = ""
		config.DocsPath = ""
		config.SchemasPath = ""
	}
	servidor := humago.New(mux, config)
	huma.Register(servidor, huma.Operation{OperationID: "estado-catalogo", Method: http.MethodGet, Path: "/api/estado", Summary: "Estado de la plantilla, sin datos de negocio"}, func(ctx context.Context, entrada *struct{}) (*Salida, error) {
		return &Salida{Body: service.ConsultarEstado()}, nil
	})

	// Conectar rutas del catálogo
	RegistrarCatalogo(servidor)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2_000_000_000)
		defer cancel()
		if err := db.Disponible(ctx); err != nil {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"title":"Base de datos no disponible","status":503}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return servidor
}
