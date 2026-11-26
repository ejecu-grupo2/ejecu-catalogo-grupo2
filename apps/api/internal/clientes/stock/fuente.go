package stock

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/sesion"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Fuente adapta el cliente generado (stock.gen.go) a service.FuenteStock. Traduce las respuestas de Stock a los errores del service y
// reenvía el JWT y X-Request-Id de la request original.
type Fuente struct {
	cliente *ClientWithResponses
}

// NuevaFuente crea el cliente con timeout explícito (5 s según el DAS).
func NuevaFuente(urlBase string, timeout time.Duration) (*Fuente, error) {
	cliente, err := NewClientWithResponses(urlBase, WithHTTPClient(&http.Client{Timeout: timeout}))
	if err != nil {
		return nil, err
	}
	return &Fuente{cliente: cliente}, nil
}

// StockDe consulta GET /api/stock/productos/{producto_id}.
func (f *Fuente) StockDe(ctx context.Context, productoID string) (service.Existencia, error) {
	var id openapi_types.UUID
	if err := id.UnmarshalText([]byte(productoID)); err != nil {
		return service.Existencia{}, fmt.Errorf("%w: id %q no es UUID", service.ErrProductoNoEncontrado, productoID)
	}
	respuesta, err := f.cliente.ObtenerStockProductoWithResponse(
		ctx, id, &ObtenerStockProductoParams{}, sesion.Reenviar,
	)
	if err != nil {
		return service.Existencia{}, fmt.Errorf("%w: %v", service.ErrDependenciaNoDisponible, err)
	}
	switch respuesta.StatusCode() {
	case http.StatusOK:
		if respuesta.JSON200 == nil {
			return service.Existencia{}, fmt.Errorf("%w: Stock respondió 200 sin cuerpo JSON", service.ErrDependenciaNoDisponible)
		}
		cuerpo := respuesta.JSON200
		if cuerpo.StockActual == nil || cuerpo.ActualizadoEn == nil {
			return service.Existencia{}, fmt.Errorf("%w: Stock respondió 200 sin stock_actual o actualizado_en", service.ErrDependenciaNoDisponible)
		}
		return service.Existencia{
			Stock:         *cuerpo.StockActual,
			ActualizadoEn: *cuerpo.ActualizadoEn,
		}, nil
	case http.StatusNotFound:
		return service.Existencia{}, service.ErrProductoNoEncontrado
	case http.StatusUnauthorized:
		return service.Existencia{}, service.ErrNoAutenticado
	case http.StatusForbidden:
		return service.Existencia{}, service.ErrSinPermiso
	default:
		return service.Existencia{}, fmt.Errorf("%w: Stock respondió %d", service.ErrDependenciaNoDisponible, respuesta.StatusCode())
	}
}
