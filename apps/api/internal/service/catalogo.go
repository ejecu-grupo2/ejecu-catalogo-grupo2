package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Estado struct {
	Servicio string `json:"servicio"`
	Estado   string `json:"estado"`
}

func ConsultarEstado() Estado { return Estado{Servicio: "catalogo", Estado: "base preparada"} }

var (
	// ErrProductoNoEncontrado: el producto no existe, no se puede vender o no tiene stock.
	ErrProductoNoEncontrado = errors.New("producto no encontrado en el catálogo")
	// ErrDependenciaNoDisponible: Productos o Stock no respondió a tiempo o respondió con error.
	ErrDependenciaNoDisponible = errors.New("servicio dependiente no disponible")
	// ErrNoAutenticado: el servicio dependiente rechazó las credenciales (401).
	ErrNoAutenticado = errors.New("credenciales ausentes o inválidas")
	// ErrSinPermiso: el usuario no tiene el permiso que pide el servicio dependiente (403).
	ErrSinPermiso = errors.New("sin permiso para la operación")
)

// ItemCatalogo representa un producto listo para ser comercializado, combinando
// los datos base del producto con su disponibilidad en inventario.
type ItemCatalogo struct {
	ID            string    `json:"id" format:"uuid" example:"3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60" doc:"Identificador único del producto (UUID)"`
	SKU           string    `json:"sku" example:"ELEC-001" doc:"Código único de referencia"`
	Nombre        string    `json:"nombre" example:"Teclado Mecánico RGB" doc:"Nombre comercial"`
	Descripcion   string    `json:"descripcion" example:"Switches táctiles y retroiluminación" doc:"Detalle del producto"`
	CategoriaIDs  []string  `json:"categoria_ids" nullable:"false" example:"[\"6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e\"]" doc:"Identificadores UUID de las categorías a las que pertenece"`
	Categoria     string    `json:"categoria" example:"Periféricos" deprecated:"true" doc:"Obsoleto: usar categoria_ids. Nombre de la categoría; el Servicio de Productos no lo entrega"`
	Precio        int64     `json:"precio" example:"49990" doc:"Precio final de venta en pesos (CLP), IVA incluido"`
	Moneda        string    `json:"moneda" example:"CLP" doc:"Moneda en formato ISO 4217"`
	Stock         int       `json:"stock" example:"15" doc:"Unidades disponibles en inventario"`
	Disponible    bool      `json:"disponible" example:"true" doc:"Indica si tiene stock para compra inmediata"`
	ActualizadoEn time.Time `json:"actualizado_en" doc:"Última actualización de datos"`
}

// Categoria representa las categorías disponibles en el catálogo.
type Categoria struct {
	ID     string `json:"id" format:"uuid" example:"6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e" doc:"Identificador de la categoría (UUID)"`
	Nombre string `json:"nombre" example:"Periféricos" doc:"Nombre visible"`
}

// ProductoBase son los datos que entrega el Servicio de Productos (grupo 3).
// Los montos van en pesos chilenos (CLP).
type ProductoBase struct {
	ID           string
	SKU          string
	Nombre       string
	Descripcion  string
	CategoriaIDs []string
	// Categoria es el nombre de la categoría. Productos no lo entrega: hoy solo lo
	// llena ProductosSimulados y en el contrato del catálogo está obsoleto.
	Categoria string
	// Precio es el costo_unitario de Productos, que es el precio de venta con IVA.
	Precio       int64
	PermiteVenta bool
}

// FiltroProductos son los filtros que acepta el listado del Servicio de Productos
// (GET /api/productos). Un campo vacío o en false no filtra.
type FiltroProductos struct {
	CategoriaID string
	// SoloVendibles pide solo los productos con permite_venta = true.
	SoloVendibles bool
}

// Filtro son los filtros de la consulta del catálogo.
type Filtro struct {
	// CategoriaID filtra por el UUID de una categoría.
	CategoriaID string
	// Categoria filtra por nombre. Obsoleto: Productos no entrega nombres de
	// categoría, así que este filtro se aplica solo en el catálogo.
	Categoria string
}

// Existencia es el stock de un producto según el Servicio de Stock (grupo 8).
type Existencia struct {
	Stock         int64
	ActualizadoEn time.Time
}

// FuenteProductos entrega los productos y categorías. Hoy la implementa
// ProductosSimulados; en el Hito 2, el cliente del Servicio de Productos.
type FuenteProductos interface {
	ListarProductos(ctx context.Context, filtro FiltroProductos) ([]ProductoBase, error)
	ObtenerProducto(ctx context.Context, id string) (ProductoBase, error)
	ListarCategorias(ctx context.Context) ([]Categoria, error)
}

// FuenteStock entrega el stock de un producto. La implementa el cliente del
// Servicio de Stock (internal/clientes/stock). Si el producto no existe en
// Stock devuelve ErrProductoNoEncontrado.
type FuenteStock interface {
	StockDe(ctx context.Context, productoID string) (Existencia, error)
}

// Pagina indica qué parte de la lista se pide. Numero parte en 1.
type Pagina struct {
	Numero int
	Tamano int
}

// Resultado es una página del catálogo y el total de productos vendibles.
type Resultado struct {
	Items []ItemCatalogo
	Total int
}

// Catalogo combina Productos y Stock. No conoce HTTP ni SQL.
type Catalogo struct {
	productos    FuenteProductos
	stock        FuenteStock
	concurrencia int // concurrencia debe ser mayor que 0: con 0 existencias se bloquea y con un valor negativo hace un panic
}

// NuevoCatalogo arma el service con sus dos fuentes.
func NuevoCatalogo(productos FuenteProductos, stock FuenteStock) *Catalogo {
	return &Catalogo{productos: productos, stock: stock, concurrencia: 8}
}

// Consultar devuelve los productos vendibles (permite_venta y stock mayor a 0), filtrados por categoría si se indica, y la página pedida.
func (c *Catalogo) Consultar(ctx context.Context, filtro Filtro, pagina Pagina) (Resultado, error) {
	productos, err := c.productos.ListarProductos(ctx, FiltroProductos{CategoriaID: filtro.CategoriaID, SoloVendibles: true})
	if err != nil {
		return Resultado{}, err
	}
	candidatos := make([]ProductoBase, 0, len(productos))
	for _, p := range productos {
		// permite_venta se revisa aunque Productos ya lo filtre: un producto que no
		// se puede vender nunca debe llegar al catálogo (modo de falla 5).
		if p.PermiteVenta && (filtro.Categoria == "" || p.Categoria == filtro.Categoria) {
			candidatos = append(candidatos, p)
		}
	}
	ids := make([]string, len(candidatos))
	for i, p := range candidatos {
		ids[i] = p.ID
	}
	existencias, err := c.existencias(ctx, ids)
	if err != nil {
		return Resultado{}, err
	}
	vendibles := make([]ItemCatalogo, 0, len(candidatos))
	for _, p := range candidatos {
		if e, ok := existencias[p.ID]; ok && e.Stock > 0 {
			vendibles = append(vendibles, armarItem(p, e))
		}
	}
	return Resultado{Items: recortar(vendibles, pagina), Total: len(vendibles)}, nil
}

// Obtener devuelve un producto vendible. Si no existe, no se puede vender o no tiene stock, devuelve ErrProductoNoEncontrado.
func (c *Catalogo) Obtener(ctx context.Context, id string) (ItemCatalogo, error) {
	p, err := c.productos.ObtenerProducto(ctx, id)
	if err != nil {
		return ItemCatalogo{}, err
	}
	if !p.PermiteVenta {
		return ItemCatalogo{}, ErrProductoNoEncontrado
	}
	e, err := c.stock.StockDe(ctx, id)
	if err != nil {
		return ItemCatalogo{}, err
	}
	if e.Stock <= 0 {
		return ItemCatalogo{}, ErrProductoNoEncontrado
	}
	return armarItem(p, e), nil
}

// Categorias devuelve las categorías del catálogo.
func (c *Catalogo) Categorias(ctx context.Context) ([]Categoria, error) {
	return c.productos.ListarCategorias(ctx)
}

// existencias consulta el stock de varios productos en paralelo.
// STOCK NO TIENE UN ENDPOINT PARA CONSULTAR POR TODOS LOS PRODUCTOS.
// Los productos que Stock no conoce se omiten.
// Ante el primer error distinto de "no encontrado" se cancelan las demás llamadas y se devuelve ese error.
func (c *Catalogo) existencias(ctx context.Context, ids []string) (map[string]Existencia, error) {
	ctxLlamadas, cancelar := context.WithCancel(ctx)
	defer cancelar()

	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		primerError error
		resultado   = make(map[string]Existencia, len(ids))
		limite      = make(chan struct{}, c.concurrencia)
	)
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case limite <- struct{}{}:
				defer func() { <-limite }()
			case <-ctxLlamadas.Done():
				return
			}
			e, err := c.stock.StockDe(ctxLlamadas, id)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				resultado[id] = e
			case errors.Is(err, ErrProductoNoEncontrado):
			case primerError == nil:
				primerError = err
				cancelar()
			}
		}()
	}
	wg.Wait()
	if primerError != nil {
		return nil, primerError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return resultado, nil
}

// monedaCLP es la moneda de todos los montos: Productos los entrega en pesos chilenos.
const monedaCLP = "CLP"

func armarItem(p ProductoBase, e Existencia) ItemCatalogo {
	return ItemCatalogo{
		ID:            p.ID,
		SKU:           p.SKU,
		Nombre:        p.Nombre,
		Descripcion:   p.Descripcion,
		CategoriaIDs:  append([]string{}, p.CategoriaIDs...), // nunca null en el JSON
		Categoria:     p.Categoria,
		Precio:        p.Precio,
		Moneda:        monedaCLP,
		Stock:         int(e.Stock),
		Disponible:    e.Stock > 0,
		ActualizadoEn: e.ActualizadoEn,
	}
}

func recortar(items []ItemCatalogo, pagina Pagina) []ItemCatalogo {
	if pagina.Numero < 1 || pagina.Tamano < 1 {
		return items
	}
	inicio := (pagina.Numero - 1) * pagina.Tamano
	if inicio >= len(items) {
		return []ItemCatalogo{}
	}
	return items[inicio:min(inicio+pagina.Tamano, len(items))]
}
