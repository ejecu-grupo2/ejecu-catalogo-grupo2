package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/api"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/sesion"
	"github.com/danielgtaylor/huma/v2/humatest"
)

const idTeclado = "3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60"

// stockFijo simula el Servicio de Stock y guarda las credenciales que recibió.
type stockFijo struct {
	mu        sync.Mutex
	err       error
	recibidas sesion.Credenciales
}

func (s *stockFijo) StockDe(ctx context.Context, id string) (service.Existencia, error) {
	s.mu.Lock()
	s.recibidas = sesion.De(ctx)
	s.mu.Unlock()
	if s.err != nil {
		return service.Existencia{}, s.err
	}
	return service.Existencia{Stock: 10, ActualizadoEn: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}, nil
}

func nuevaAPI(t *testing.T, stock service.FuenteStock) humatest.TestAPI {
	t.Helper()
	_, humaAPI := humatest.New(t)
	api.RegistrarCatalogo(humaAPI, service.NuevoCatalogo(service.ProductosSimulados{}, stock))
	return humaAPI
}

func esperarEstado(t *testing.T, resp *httptest.ResponseRecorder, estado int) {
	t.Helper()
	if resp.Code != estado {
		t.Fatalf("se esperaba status %d, obtenido %d: %s", estado, resp.Code, resp.Body.String())
	}
}

func TestCatalogoEndpoints(t *testing.T) {
	humaAPI := nuevaAPI(t, &stockFijo{})

	t.Run("GET /api/catalogo responde 200 con productos y total", func(t *testing.T) {
		resp := humaAPI.Get("/api/catalogo")
		esperarEstado(t, resp, http.StatusOK)
		var items []service.ItemCatalogo
		if err := json.Unmarshal(resp.Body.Bytes(), &items); err != nil || len(items) != 3 {
			t.Fatalf("se esperaban 3 productos: %d, err=%v", len(items), err)
		}
		if total := resp.Header().Get("X-Total-Count"); total != "3" {
			t.Fatalf("X-Total-Count=%q, se esperaba 3", total)
		}
	})

	t.Run("GET /api/catalogo pagina los resultados", func(t *testing.T) {
		resp := humaAPI.Get("/api/catalogo?pagina=2&tamano=2")
		esperarEstado(t, resp, http.StatusOK)
		var items []service.ItemCatalogo
		if err := json.Unmarshal(resp.Body.Bytes(), &items); err != nil || len(items) != 1 {
			t.Fatalf("la página 2 de tamaño 2 debe traer 1 producto: %d, err=%v", len(items), err)
		}
		if total := resp.Header().Get("X-Total-Count"); total != "3" {
			t.Fatalf("X-Total-Count=%q, se esperaba 3", total)
		}
	})

	t.Run("GET /api/catalogo rechaza un tamaño mayor a 100", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo?tamano=101"), http.StatusUnprocessableEntity)
	})

	t.Run("GET /api/catalogo con filtro obsoleto por nombre responde 200", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo?categoria=Periféricos"), http.StatusOK)
	})

	t.Run("GET /api/catalogo filtra por categoria_id", func(t *testing.T) {
		resp := humaAPI.Get("/api/catalogo?categoria_id=1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b")
		esperarEstado(t, resp, http.StatusOK)
		var items []service.ItemCatalogo
		if err := json.Unmarshal(resp.Body.Bytes(), &items); err != nil || len(items) != 1 || items[0].SKU != "ACC-001" {
			t.Fatalf("se esperaba solo el hub de Accesorios: %+v, err=%v", items, err)
		}
	})

	t.Run("GET /api/catalogo responde 422 si categoria_id no es UUID", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo?categoria_id=perifericos"), http.StatusUnprocessableEntity)
	})

	t.Run("GET /api/catalogo/{id} responde 200 si existe", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo/"+idTeclado), http.StatusOK)
	})

	t.Run("GET /api/catalogo/{id} responde 404 si no existe", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo/00000000-0000-4000-8000-000000000000"), http.StatusNotFound)
	})

	t.Run("GET /api/catalogo/{id} responde 422 si el id no es UUID", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/catalogo/prod-101"), http.StatusUnprocessableEntity)
	})

	t.Run("GET /api/categorias responde 200 con categorias", func(t *testing.T) {
		esperarEstado(t, humaAPI.Get("/api/categorias"), http.StatusOK)
	})
}

func TestCatalogoTraduceErroresDeStock(t *testing.T) {
	casos := []struct {
		err    error
		estado int
	}{
		{fmt.Errorf("%w: timeout", service.ErrDependenciaNoDisponible), http.StatusServiceUnavailable},
		{service.ErrNoAutenticado, http.StatusUnauthorized},
		{service.ErrSinPermiso, http.StatusForbidden},
	}
	for _, caso := range casos {
		t.Run(http.StatusText(caso.estado), func(t *testing.T) {
			humaAPI := nuevaAPI(t, &stockFijo{err: caso.err})
			for _, ruta := range []string{"/api/catalogo", "/api/catalogo/" + idTeclado} {
				resp := humaAPI.Get(ruta)
				esperarEstado(t, resp, caso.estado)
				if tipo := resp.Header().Get("Content-Type"); !strings.HasPrefix(tipo, "application/problem+json") {
					t.Fatalf("%s: Content-Type=%q", ruta, tipo)
				}
				if strings.Contains(resp.Body.String(), "timeout") {
					t.Fatalf("%s: la respuesta expone el detalle interno: %s", ruta, resp.Body.String())
				}
			}
		})
	}
}

type baseDisponible struct{}

func (baseDisponible) Disponible(context.Context) error { return nil }

func TestRegistrarReenviaCredencialesYRequestID(t *testing.T) {
	stock := &stockFijo{}
	mux := http.NewServeMux()
	api.Registrar(mux, "test", baseDisponible{}, service.NuevoCatalogo(service.ProductosSimulados{}, stock))

	t.Run("desde la cookie, generando el X-Request-Id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/catalogo/"+idTeclado, nil)
		req.AddCookie(&http.Cookie{Name: sesion.NombreCookie, Value: "jwt-navegador"})
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		esperarEstado(t, resp, http.StatusOK)
		generado := resp.Header().Get("X-Request-Id")
		if generado == "" {
			t.Fatal("la API debe generar un X-Request-Id si no viene")
		}
		if stock.recibidas.Token != "jwt-navegador" || stock.recibidas.RequestID != generado {
			t.Fatalf("credenciales recibidas por Stock: %+v", stock.recibidas)
		}
	})

	t.Run("desde Authorization, propagando el X-Request-Id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/catalogo", nil)
		req.Header.Set("Authorization", "Bearer jwt-servicio")
		req.Header.Set("X-Request-Id", "req-externo")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		esperarEstado(t, resp, http.StatusOK)
		if resp.Header().Get("X-Request-Id") != "req-externo" {
			t.Fatalf("X-Request-Id devuelto: %q", resp.Header().Get("X-Request-Id"))
		}
		if stock.recibidas.Token != "jwt-servicio" || stock.recibidas.RequestID != "req-externo" {
			t.Fatalf("credenciales recibidas por Stock: %+v", stock.recibidas)
		}
	})
}

func TestContratoMarcaCategoriaComoObsoleta(t *testing.T) {
	humaAPI := nuevaAPI(t, &stockFijo{})
	operacion := humaAPI.OpenAPI().Paths["/api/catalogo"].Get
	obsoletos := map[string]bool{}
	for _, p := range operacion.Parameters {
		obsoletos[p.Name] = p.Deprecated
	}
	if !obsoletos["categoria"] || obsoletos["categoria_id"] {
		t.Fatalf("parámetros obsoletos: %v; solo categoria debe estarlo", obsoletos)
	}
}
