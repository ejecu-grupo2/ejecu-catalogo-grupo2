package service

import (
	"context"
	"errors"
	"time"
)

type Estado struct {
	Servicio string `json:"servicio"`
	Estado   string `json:"estado"`
}

func ConsultarEstado() Estado { return Estado{Servicio: "catalogo", Estado: "base preparada"} }

var (
	ErrProductoNoEncontrado = errors.New("producto no encontrado en el catálogo")
)

// ItemCatalogo representa un producto listo para ser comercializado, combinando
// los datos base del producto con su disponibilidad en inventario.
type ItemCatalogo struct {
	ID            string    `json:"id" example:"prod-101" doc:"Identificador único del producto"`
	SKU           string    `json:"sku" example:"ELEC-001" doc:"Código único de referencia"`
	Nombre        string    `json:"nombre" example:"Teclado Mecánico RGB" doc:"Nombre comercial"`
	Descripcion   string    `json:"descripcion" example:"Switches táctiles y retroiluminación" doc:"Detalle del producto"`
	Categoria     string    `json:"categoria" example:"Periféricos" doc:"Categoría a la que pertenece"`
	Precio        int64     `json:"precio" example:"49990" doc:"Precio final de venta en pesos (CLP)"`
	Moneda        string    `json:"moneda" example:"CLP" doc:"Moneda en formato ISO 4217"`
	Stock         int       `json:"stock" example:"15" doc:"Unidades disponibles en inventario"`
	Disponible    bool      `json:"disponible" example:"true" doc:"Indica si tiene stock para compra inmediata"`
	ActualizadoEn time.Time `json:"actualizado_en" doc:"Última actualización de datos"`
}

// Categoria representa las categorías disponibles en el catálogo.
type Categoria struct {
	ID     string `json:"id" example:"perifericos" doc:"Identificador de la categoría"`
	Nombre string `json:"nombre" example:"Periféricos" doc:"Nombre visible"`
}

// Datos simulados (mocks) que integran conceptualmente datos de Productos (G3) y Stock (G8)
var itemsCatalogoMock = []ItemCatalogo{
	{
		ID:            "prod-101",
		SKU:           "ELEC-001",
		Nombre:        "Teclado Mecánico RGB",
		Descripcion:   "Teclado retroiluminado con switches táctiles y conexión USB-C",
		Categoria:     "Periféricos",
		Precio:        49990,
		Moneda:        "CLP",
		Stock:         15,
		Disponible:    true,
		ActualizadoEn: time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC),
	},
	{
		ID:            "prod-102",
		SKU:           "ELEC-002",
		Nombre:        "Mouse Inalámbrico Ergonómico",
		Descripcion:   "Sensor óptico de alta precisión con conectividad Bluetooth",
		Categoria:     "Periféricos",
		Precio:        29990,
		Moneda:        "CLP",
		Stock:         32,
		Disponible:    true,
		ActualizadoEn: time.Date(2026, 11, 8, 15, 30, 0, 0, time.UTC),
	},
	{
		ID:            "prod-103",
		SKU:           "ACC-001",
		Nombre:        "Hub USB-C 7 en 1",
		Descripcion:   "Adaptador multipuerto con HDMI 4K y puertos USB 3.0",
		Categoria:     "Accesorios",
		Precio:        21990,
		Moneda:        "CLP",
		Stock:         8,
		Disponible:    true,
		ActualizadoEn: time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC),
	},
}

var categoriasMock = []Categoria{
	{ID: "perifericos", Nombre: "Periféricos"},
	{ID: "accesorios", Nombre: "Accesorios"},
}

// ConsultarCatalogo retorna la lista de productos vendibles con filtro opcional de categoría.
func ConsultarCatalogo(ctx context.Context, categoria string) ([]ItemCatalogo, error) {
	if categoria == "" {
		return itemsCatalogoMock, nil
	}
	filtrados := make([]ItemCatalogo, 0)
	for _, item := range itemsCatalogoMock {
		if item.Categoria == categoria {
			filtrados = append(filtrados, item)
		}
	}
	return filtrados, nil
}

// ObtenerItemCatalogo retorna el detalle de un producto vendible por ID.
func ObtenerItemCatalogo(ctx context.Context, id string) (ItemCatalogo, error) {
	for _, item := range itemsCatalogoMock {
		if item.ID == id {
			return item, nil
		}
	}
	return ItemCatalogo{}, ErrProductoNoEncontrado
}

// ConsultarCategorias retorna la lista de categorías existentes en el catálogo.
func ConsultarCategorias(ctx context.Context) ([]Categoria, error) {
	return categoriasMock, nil
}
