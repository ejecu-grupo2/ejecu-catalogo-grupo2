package service

import (
	"context"
	"slices"
)

// ProductosSimulados reemplaza al Servicio de Productos (grupo 3) mientras no
// esté su cliente. Sigue la forma de su contrato y los UUID son fijos para las pruebas.
type ProductosSimulados struct{}

var productosSimulados = []ProductoBase{
	{
		ID:           "3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60",
		SKU:          "ELEC-001",
		Nombre:       "Teclado Mecánico RGB",
		Descripcion:  "Teclado retroiluminado con switches táctiles y conexión USB-C",
		CategoriaIDs: []string{"6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e"},
		Categoria:    "Periféricos",
		Precio:       49990,
		PermiteVenta: true,
	},
	{
		ID:           "8a1d4e7b-2c3f-4a5b-8c9d-0e1f2a3b4c5d",
		SKU:          "ELEC-002",
		Nombre:       "Mouse Inalámbrico Ergonómico",
		Descripcion:  "Sensor óptico de alta precisión con conectividad Bluetooth",
		CategoriaIDs: []string{"6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e"},
		Categoria:    "Periféricos",
		Precio:       29990,
		PermiteVenta: true,
	},
	{
		ID:           "c5e8f1a2-9b3c-4d6e-a7f8-2b3c4d5e6f70",
		SKU:          "ACC-001",
		Nombre:       "Hub USB-C 7 en 1",
		Descripcion:  "Adaptador multipuerto con HDMI 4K y puertos USB 3.0",
		CategoriaIDs: []string{"1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b"},
		Categoria:    "Accesorios",
		Precio:       21990,
		PermiteVenta: true,
	},
}

var categoriasSimuladas = []Categoria{
	{ID: "6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e", Nombre: "Periféricos"},
	{ID: "1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b", Nombre: "Accesorios"},
}

// ListarProductos aplica los filtros como lo hace GET /api/productos.
func (ProductosSimulados) ListarProductos(_ context.Context, filtro FiltroProductos) ([]ProductoBase, error) {
	productos := make([]ProductoBase, 0, len(productosSimulados))
	for _, p := range productosSimulados {
		if filtro.SoloVendibles && !p.PermiteVenta {
			continue
		}
		if filtro.CategoriaID != "" && !slices.Contains(p.CategoriaIDs, filtro.CategoriaID) {
			continue
		}
		productos = append(productos, p)
	}
	return productos, nil
}

func (ProductosSimulados) ObtenerProducto(_ context.Context, id string) (ProductoBase, error) {
	for _, p := range productosSimulados {
		if p.ID == id {
			return p, nil
		}
	}
	return ProductoBase{}, ErrProductoNoEncontrado
}

func (ProductosSimulados) ListarCategorias(context.Context) ([]Categoria, error) {
	return categoriasSimuladas, nil
}
