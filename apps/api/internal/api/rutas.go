package api

import (
	"context"
	"net/http"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/sesion"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type Disponibilidad interface{ Disponible(context.Context) error }
type Salida struct{ Body service.Estado }

func Registrar(mux *http.ServeMux, entorno string, db Disponibilidad, catalogo *service.Catalogo) huma.API {
	config := huma.DefaultConfig("Catálogo", "0.1.0")
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	config.SchemasPath = "/api/schemas"
	if entorno == "prod" {
		config.OpenAPIPath = ""
		config.DocsPath = ""
		config.SchemasPath = ""
	}
	// Se toma como base el contrato de Stock, JWT en cookie desde el navegador, o como Bearer cuando lo reenvía otro servicio.
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"cookieAuth": {
			Type:        "apiKey",
			In:          "cookie",
			Name:        sesion.NombreCookie,
			Description: "JWT emitido por el servicio de identidad en cookie HttpOnly, Secure, SameSite=Lax. Confirmar el nombre de la cookie.",
		},
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Mismo JWT, reenviado entre servicios.",
		},
	}
	servidor := humago.New(mux, config)
	// Debe registrarse antes que las operaciones: Huma toma los middlewares al registrar.
	servidor.UseMiddleware(credenciales)

	huma.Register(servidor, huma.Operation{OperationID: "estado-catalogo", Method: http.MethodGet, Path: "/api/estado", Summary: "Estado de la plantilla, sin datos de negocio"}, func(ctx context.Context, entrada *struct{}) (*Salida, error) {
		return &Salida{Body: service.ConsultarEstado()}, nil
	})

	// Conectar rutas del catálogo
	RegistrarCatalogo(servidor, catalogo)

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

// credenciales guarda el JWT y el X-Request-Id de la request en el contexto, para
// que los clientes de Stock y Productos los reenvíen. Si no viene X-Request-Id,
// genera uno y lo devuelve en la respuesta.
func credenciales(ctx huma.Context, next func(huma.Context)) {
	c := sesion.Credenciales{
		Token:     sesion.TokenDe(ctx.Header("Authorization"), ctx.Header("Cookie")),
		RequestID: ctx.Header("X-Request-Id"),
	}
	if c.RequestID == "" {
		c.RequestID = sesion.NuevoRequestID()
	}
	ctx.SetHeader("X-Request-Id", c.RequestID)
	next(huma.WithContext(ctx, sesion.Con(ctx.Context(), c)))
}
