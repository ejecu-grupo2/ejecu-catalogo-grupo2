package api_test

import (
	"net/http"
	"testing"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/api"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestCatalogoEndpoints(t *testing.T) {
	_, humaApi := humatest.New(t)
	api.RegistrarCatalogo(humaApi)

	t.Run("GET /api/catalogo responde 200 con productos", func(t *testing.T) {
		resp := humaApi.Get("/api/catalogo")
		if resp.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200, obtenido %d", resp.Code)
		}
	})

	t.Run("GET /api/catalogo con filtro responde 200", func(t *testing.T) {
		resp := humaApi.Get("/api/catalogo?categoria=Periféricos")
		if resp.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200, obtenido %d", resp.Code)
		}
	})

	t.Run("GET /api/catalogo/{id} responde 200 si existe", func(t *testing.T) {
		resp := humaApi.Get("/api/catalogo/prod-101")
		if resp.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200, obtenido %d", resp.Code)
		}
	})

	t.Run("GET /api/catalogo/{id} responde 404 si no existe", func(t *testing.T) {
		resp := humaApi.Get("/api/catalogo/id-inexistente")
		if resp.Code != http.StatusNotFound {
			t.Fatalf("se esperaba status 404, obtenido %d", resp.Code)
		}
	})

	t.Run("GET /api/categorias responde 200 con categorias", func(t *testing.T) {
		resp := humaApi.Get("/api/categorias")
		if resp.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200, obtenido %d", resp.Code)
		}
	})
}
