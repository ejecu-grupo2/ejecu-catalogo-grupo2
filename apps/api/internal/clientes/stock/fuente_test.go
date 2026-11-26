package stock_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/clientes/stock"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/sesion"
)

const idProducto = "3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60"

func servidorStock(t *testing.T, manejador http.HandlerFunc) *stock.Fuente {
	t.Helper()
	servidor := httptest.NewServer(manejador)
	t.Cleanup(servidor.Close)
	fuente, err := stock.NuevaFuente(servidor.URL, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("no se pudo crear el cliente: %v", err)
	}
	return fuente
}

func responder(estado int, cuerpo string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tipo := "application/json"
		if estado >= 400 {
			tipo = "application/problem+json"
		}
		w.Header().Set("Content-Type", tipo)
		w.WriteHeader(estado)
		_, _ = w.Write([]byte(cuerpo))
	}
}

func TestStockDeConvierteLaRespuesta(t *testing.T) {
	var ruta, autorizacion, requestID string
	fuente := servidorStock(t, func(w http.ResponseWriter, r *http.Request) {
		ruta = r.URL.Path
		autorizacion = r.Header.Get("Authorization")
		requestID = r.Header.Get("X-Request-Id")
		responder(http.StatusOK, `{"producto_id":"`+idProducto+`","stock_actual":18,"actualizado_en":"2026-11-22T12:00:00Z"}`)(w, r)
	})
	ctx := sesion.Con(context.Background(), sesion.Credenciales{Token: "jwt-prueba", RequestID: "req-1"})

	e, err := fuente.StockDe(ctx, idProducto)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if e.Stock != 18 || !e.ActualizadoEn.Equal(time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("existencia convertida: %+v", e)
	}
	if ruta != "/api/stock/productos/"+idProducto {
		t.Fatalf("ruta llamada: %s", ruta)
	}
	if autorizacion != "Bearer jwt-prueba" || requestID != "req-1" {
		t.Fatalf("no se reenviaron las credenciales: Authorization=%q X-Request-Id=%q", autorizacion, requestID)
	}
}

func TestStockDeTraduceErrores(t *testing.T) {
	casos := []struct {
		estado   int
		esperado error
	}{
		{http.StatusNotFound, service.ErrProductoNoEncontrado},
		{http.StatusUnauthorized, service.ErrNoAutenticado},
		{http.StatusForbidden, service.ErrSinPermiso},
		{http.StatusInternalServerError, service.ErrDependenciaNoDisponible},
		{http.StatusServiceUnavailable, service.ErrDependenciaNoDisponible},
	}
	for _, caso := range casos {
		t.Run(http.StatusText(caso.estado), func(t *testing.T) {
			fuente := servidorStock(t, responder(caso.estado, fmt.Sprintf(`{"title":"error","status":%d}`, caso.estado)))
			if _, err := fuente.StockDe(context.Background(), idProducto); !errors.Is(err, caso.esperado) {
				t.Fatalf("se esperaba %v, obtenido %v", caso.esperado, err)
			}
		})
	}
}

func TestStockDeRespetaElTimeout(t *testing.T) {
	fuente := servidorStock(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	})
	inicio := time.Now()
	_, err := fuente.StockDe(context.Background(), idProducto)
	if !errors.Is(err, service.ErrDependenciaNoDisponible) {
		t.Fatalf("se esperaba ErrDependenciaNoDisponible, obtenido %v", err)
	}
	if time.Since(inicio) > time.Second {
		t.Fatalf("la llamada no se cortó en el timeout: tardó %v", time.Since(inicio))
	}
}

func TestStockDeConServicioCaido(t *testing.T) {
	servidor := httptest.NewServer(http.NotFoundHandler())
	url := servidor.URL
	servidor.Close()
	fuente, err := stock.NuevaFuente(url, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("no se pudo crear el cliente: %v", err)
	}
	if _, err := fuente.StockDe(context.Background(), idProducto); !errors.Is(err, service.ErrDependenciaNoDisponible) {
		t.Fatalf("se esperaba ErrDependenciaNoDisponible, obtenido %v", err)
	}
}

func TestStockDeConIdInvalido(t *testing.T) {
	llamado := false
	fuente := servidorStock(t, func(http.ResponseWriter, *http.Request) { llamado = true })
	if _, err := fuente.StockDe(context.Background(), "prod-101"); !errors.Is(err, service.ErrProductoNoEncontrado) {
		t.Fatalf("se esperaba ErrProductoNoEncontrado, obtenido %v", err)
	}
	if llamado {
		t.Fatal("no debe llamar a Stock con un id que no es UUID")
	}
}
