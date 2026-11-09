package api

import (
	"context"
	"net/http"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/danielgtaylor/huma/v2"
)

// EntradaConsultarCatalogo define los filtros de consulta.
type EntradaConsultarCatalogo struct {
	Categoria string `query:"categoria" required:"false" doc:"Filtrar por nombre de categoría"`
}

// SalidaConsultarCatalogo contiene la lista de ítems vendibles.
type SalidaConsultarCatalogo struct {
	Body []service.ItemCatalogo
}

// EntradaDetalleItemCatalogo define el parámetro de ruta.
type EntradaDetalleItemCatalogo struct {
	ID string `path:"id" example:"prod-101" doc:"Identificador único del producto"`
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
func RegistrarCatalogo(servidor huma.API) {
	// GET /api/catalogo
	huma.Register(servidor, huma.Operation{
		OperationID: "consultar-catalogo",
		Method:      http.MethodGet,
		Path:        "/api/catalogo",
		Summary:     "Listar productos vendibles",
		Description: "Retorna los productos con stock disponible para la venta, con filtro opcional por categoría.",
		Tags:        []string{"Catálogo"},
	}, func(ctx context.Context, entrada *EntradaConsultarCatalogo) (*SalidaConsultarCatalogo, error) {
		items, err := service.ConsultarCatalogo(ctx, entrada.Categoria)
		if err != nil {
			return nil, huma.Error500InternalServerError("Error al consultar el catálogo", err)
		}
		return &SalidaConsultarCatalogo{Body: items}, nil
	})

	// GET /api/catalogo/{id}
	huma.Register(servidor, huma.Operation{
		OperationID: "obtener-item-catalogo",
		Method:      http.MethodGet,
		Path:        "/api/catalogo/{id}",
		Summary:     "Detalle de un producto en catálogo",
		Description: "Obtiene los detalles comerciales y disponibilidad de inventario de un producto específico.",
		Tags:        []string{"Catálogo"},
	}, func(ctx context.Context, entrada *EntradaDetalleItemCatalogo) (*SalidaDetalleItemCatalogo, error) {
		item, err := service.ObtenerItemCatalogo(ctx, entrada.ID)
		if err != nil {
			if err == service.ErrProductoNoEncontrado {
				return nil, huma.Error404NotFound("Producto no encontrado en el catálogo")
			}
			return nil, huma.Error500InternalServerError("Error al obtener detalle del producto", err)
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
	}, func(ctx context.Context, entrada *struct{}) (*SalidaConsultarCategorias, error) {
		categorias, err := service.ConsultarCategorias(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Error al consultar categorías", err)
		}
		return &SalidaConsultarCategorias{Body: categorias}, nil
	})
}
