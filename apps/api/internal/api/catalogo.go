package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/danielgtaylor/huma/v2"
)

// Permisos del catálogo. Suponiendo que la validacion queda para el hito2, qeuda declarodo en el contrato igualmente
const (
	permisoOperar = "catalogo:operar"
)

// EntradaConsultarCatalogo define los filtros y la paginación.
type EntradaConsultarCatalogo struct {
	CategoriaID string `query:"categoria_id" required:"false" format:"uuid" example:"6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e" doc:"Filtrar por categoría (UUID)"`
	Categoria   string `query:"categoria" required:"false" deprecated:"true" doc:"Obsoleto: usar categoria_id. Filtrar por nombre de categoría"`
	Pagina      int    `query:"pagina" minimum:"1" default:"1" doc:"Número de página, desde 1"`
	Tamano      int    `query:"tamano" minimum:"1" maximum:"100" default:"20" doc:"Productos por página"`
}

// SalidaConsultarCatalogo contiene la página de ítems vendibles y el total.
type SalidaConsultarCatalogo struct {
	Total int `header:"X-Total-Count" doc:"Cantidad total de productos vendibles que cumplen el filtro"`
	Body  []service.ItemCatalogo
}

// EntradaDetalleItemCatalogo define el parámetro de ruta.
type EntradaDetalleItemCatalogo struct {
	ID string `path:"id" format:"uuid" example:"3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60" doc:"Identificador único del producto (UUID)"`
}

// SalidaDetalleItemCatalogo devuelve el detalle del producto.
type SalidaDetalleItemCatalogo struct {
	Body service.ItemCatalogo
}

// SalidaConsultarCategorias devuelve la lista de categorías.
type SalidaConsultarCategorias struct {
	Body []service.Categoria
}

// RegistrarCatalogo publica las rutas de consulta del catálogo vendible.
func RegistrarCatalogo(servidor huma.API, catalogo *service.Catalogo) {
	// GET /api/catalogo
	huma.Register(servidor, huma.Operation{
		OperationID: "consultar-catalogo",
		Method:      http.MethodGet,
		Path:        "/api/catalogo",
		Summary:     "Listar productos vendibles",
		Description: "Retorna los productos que se pueden vender y tienen stock, según el Servicio de Stock, con filtro opcional por categoría y paginación. El total va en el encabezado X-Total-Count.",
		Tags:        []string{"Catálogo"},
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable},
		Extensions:  map[string]any{"x-permiso": permisoOperar},
	}, func(ctx context.Context, entrada *EntradaConsultarCatalogo) (*SalidaConsultarCatalogo, error) {
		filtro := service.Filtro{CategoriaID: entrada.CategoriaID, Categoria: entrada.Categoria}
		resultado, err := catalogo.Consultar(ctx, filtro, service.Pagina{Numero: entrada.Pagina, Tamano: entrada.Tamano})
		if err != nil {
			return nil, errorHTTP(ctx, err, "Error al consultar el catálogo")
		}
		return &SalidaConsultarCatalogo{Total: resultado.Total, Body: resultado.Items}, nil
	})
	marcarParametroObsoleto(servidor, "/api/catalogo", "categoria")

	// GET /api/catalogo/{id}
	huma.Register(servidor, huma.Operation{
		OperationID: "obtener-item-catalogo",
		Method:      http.MethodGet,
		Path:        "/api/catalogo/{id}",
		Summary:     "Detalle de un producto en catálogo",
		Description: "Obtiene los detalles comerciales y el stock de un producto vendible. Responde 404 si el producto no existe, no se puede vender o no tiene stock.",
		Tags:        []string{"Catálogo"},
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusServiceUnavailable},
		Extensions:  map[string]any{"x-permiso": permisoOperar},
	}, func(ctx context.Context, entrada *EntradaDetalleItemCatalogo) (*SalidaDetalleItemCatalogo, error) {
		item, err := catalogo.Obtener(ctx, entrada.ID)
		if err != nil {
			return nil, errorHTTP(ctx, err, "Error al obtener detalle del producto")
		}
		return &SalidaDetalleItemCatalogo{Body: item}, nil
	})

	// GET /api/categorias
	huma.Register(servidor, huma.Operation{
		OperationID: "consultar-categorias",
		Method:      http.MethodGet,
		Path:        "/api/categorias",
		Summary:     "Listar categorías del catálogo",
		Description: "Retorna las categorías de productos disponibles.",
		Tags:        []string{"Catálogo"},
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
		Extensions:  map[string]any{"x-permiso": permisoOperar},
	}, func(ctx context.Context, entrada *struct{}) (*SalidaConsultarCategorias, error) {
		categorias, err := catalogo.Categorias(ctx)
		if err != nil {
			return nil, errorHTTP(ctx, err, "Error al consultar categorías")
		}
		return &SalidaConsultarCategorias{Body: categorias}, nil
	})
}

// marcarParametroObsoleto marca un parámetro como deprecated en el contrato. El tag
// deprecated de Huma solo lo marca en el schema del parámetro, no en el parámetro.
func marcarParametroObsoleto(servidor huma.API, ruta, nombre string) {
	for _, p := range servidor.OpenAPI().Paths[ruta].Get.Parameters {
		if p.Name == nombre {
			p.Deprecated = true
		}
	}
}

// errorHTTP traduce los errores del service a RFC 9457 sin exponer el detalle
// interno, que queda solo en el log (modo de falla 8).
func errorHTTP(ctx context.Context, err error, mensaje string) error {
	switch {
	case errors.Is(err, service.ErrProductoNoEncontrado):
		return huma.Error404NotFound("Producto no encontrado en el catálogo")
	case errors.Is(err, service.ErrNoAutenticado):
		return huma.Error401Unauthorized("Sesión ausente o inválida")
	case errors.Is(err, service.ErrSinPermiso):
		return huma.Error403Forbidden("Sin permiso para consultar el stock del catálogo")
	case errors.Is(err, service.ErrDependenciaNoDisponible):
		slog.WarnContext(ctx, mensaje, "error", err)
		return huma.Error503ServiceUnavailable("El Servicio de Stock no está disponible")
	default:
		slog.ErrorContext(ctx, mensaje, "error", err)
		return huma.Error500InternalServerError(mensaje)
	}
}
