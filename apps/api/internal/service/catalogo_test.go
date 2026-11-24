package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
)

const (
	idPerifericos = "6b7c8d9e-0f1a-4b2c-9d3e-4f5a6b7c8d9e"
	idAccesorios  = "1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b"

	idTeclado  = "3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60"
	idMouse    = "8a1d4e7b-2c3f-4a5b-8c9d-0e1f2a3b4c5d"
	idHub      = "c5e8f1a2-9b3c-4d6e-a7f8-2b3c4d5e6f70"
	idNoExiste = "00000000-0000-4000-8000-000000000000"
)

// stockFijo simula el Servicio de Stock. Un id ausente del mapa responde
// "no encontrado"; si err no es nil, todas las llamadas fallan con ese error.
type stockFijo struct {
	stock    map[string]int64
	err      error
	llamadas atomic.Int32
}

func (s *stockFijo) StockDe(_ context.Context, id string) (service.Existencia, error) {
	s.llamadas.Add(1)
	if s.err != nil {
		return service.Existencia{}, s.err
	}
	n, ok := s.stock[id]
	if !ok {
		return service.Existencia{}, service.ErrProductoNoEncontrado
	}
	return service.Existencia{Stock: n, ActualizadoEn: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}, nil
}

func stockCompleto() *stockFijo {
	return &stockFijo{stock: map[string]int64{idTeclado: 15, idMouse: 32, idHub: 8}}
}

func TestConsultarDevuelveSoloVendibles(t *testing.T) {
	stock := stockCompleto()
	stock.stock[idMouse] = 0
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stock)

	r, err := catalogo.Consultar(context.Background(), service.Filtro{}, service.Pagina{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if r.Total != 2 || len(r.Items) != 2 {
		t.Fatalf("se esperaban 2 vendibles, total=%d items=%d", r.Total, len(r.Items))
	}
	for _, item := range r.Items {
		if item.ID == idMouse {
			t.Fatal("un producto con stock 0 no debe aparecer en el catálogo")
		}
		if !item.Disponible || item.Stock <= 0 {
			t.Fatalf("%s aparece sin stock", item.SKU)
		}
	}
}

func TestConsultarOmiteProductosQueStockNoConoce(t *testing.T) {
	stock := stockCompleto()
	delete(stock.stock, idHub)
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stock)

	r, err := catalogo.Consultar(context.Background(), service.Filtro{}, service.Pagina{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if r.Total != 2 {
		t.Fatalf("se esperaban 2 productos, obtenido %d", r.Total)
	}
}

func TestConsultarFiltraPorCategoriaID(t *testing.T) {
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stockCompleto())

	r, err := catalogo.Consultar(context.Background(), service.Filtro{CategoriaID: idAccesorios}, service.Pagina{})
	if err != nil || r.Total != 1 || r.Items[0].ID != idHub {
		t.Fatalf("filtro Accesorios: %+v err=%v", r, err)
	}
	vacio, err := catalogo.Consultar(context.Background(), service.Filtro{CategoriaID: idNoExiste}, service.Pagina{})
	if err != nil || vacio.Items == nil || len(vacio.Items) != 0 {
		t.Fatalf("una categoría sin productos debe devolver una lista vacía: %+v err=%v", vacio, err)
	}
}

func TestConsultarFiltraPorNombreDeCategoriaYDevuelveListaVacia(t *testing.T) {
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stockCompleto())

	r, err := catalogo.Consultar(context.Background(), service.Filtro{Categoria: "Periféricos"}, service.Pagina{})
	if err != nil || r.Total != 2 {
		t.Fatalf("filtro Periféricos: total=%d err=%v", r.Total, err)
	}
	vacio, err := catalogo.Consultar(context.Background(), service.Filtro{Categoria: "Inexistente"}, service.Pagina{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if vacio.Items == nil || len(vacio.Items) != 0 {
		t.Fatal("un filtro sin resultados debe devolver una lista vacía, no nil")
	}
}

func TestConsultarPagina(t *testing.T) {
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stockCompleto())
	casos := []struct {
		pagina service.Pagina
		largo  int
	}{
		{service.Pagina{Numero: 1, Tamano: 2}, 2},
		{service.Pagina{Numero: 2, Tamano: 2}, 1},
		{service.Pagina{Numero: 3, Tamano: 2}, 0},
	}
	for _, caso := range casos {
		r, err := catalogo.Consultar(context.Background(), service.Filtro{}, caso.pagina)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if r.Total != 3 || len(r.Items) != caso.largo {
			t.Fatalf("página %+v: total=%d items=%d, se esperaba total=3 items=%d", caso.pagina, r.Total, len(r.Items), caso.largo)
		}
	}
}

func TestConsultarPropagaStockNoDisponible(t *testing.T) {
	falla := fmt.Errorf("%w: timeout", service.ErrDependenciaNoDisponible)
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, &stockFijo{err: falla})

	_, err := catalogo.Consultar(context.Background(), service.Filtro{}, service.Pagina{})
	if !errors.Is(err, service.ErrDependenciaNoDisponible) {
		t.Fatalf("se esperaba ErrDependenciaNoDisponible, obtenido %v", err)
	}
}

func TestConsultarPropagaSinPermiso(t *testing.T) {
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, &stockFijo{err: service.ErrSinPermiso})

	_, err := catalogo.Consultar(context.Background(), service.Filtro{}, service.Pagina{})
	if !errors.Is(err, service.ErrSinPermiso) {
		t.Fatalf("se esperaba ErrSinPermiso, obtenido %v", err)
	}
}

// stockLento cuenta cuántas llamadas hay en curso a la vez.
type stockLento struct {
	mu      sync.Mutex
	enCurso int
	maximo  int
}

func (s *stockLento) StockDe(ctx context.Context, _ string) (service.Existencia, error) {
	s.mu.Lock()
	s.enCurso++
	s.maximo = max(s.maximo, s.enCurso)
	s.mu.Unlock()
	time.Sleep(5 * time.Millisecond)
	s.mu.Lock()
	s.enCurso--
	s.mu.Unlock()
	return service.Existencia{Stock: 1}, nil
}

type muchosProductos struct{ service.ProductosSimulados }

func (muchosProductos) ListarProductos(context.Context, service.FiltroProductos) ([]service.ProductoBase, error) {
	productos := make([]service.ProductoBase, 40)
	for i := range productos {
		productos[i] = service.ProductoBase{ID: fmt.Sprintf("id-%02d", i), PermiteVenta: true}
	}
	return productos, nil
}

// productosSinFiltrar ignora los filtros, como un Servicio de Productos que no
// los respetara, y guarda el último filtro recibido.
type productosSinFiltrar struct {
	service.ProductosSimulados
	filtro service.FiltroProductos
}

func (p *productosSinFiltrar) ListarProductos(_ context.Context, filtro service.FiltroProductos) ([]service.ProductoBase, error) {
	p.filtro = filtro
	return []service.ProductoBase{
		{ID: idTeclado, PermiteVenta: true},
		{ID: idMouse, PermiteVenta: false},
	}, nil
}

func TestConsultarPideSoloVendiblesYLosVuelveARevisar(t *testing.T) {
	productos := &productosSinFiltrar{}
	catalogo := service.NuevoCatalogo(productos, stockCompleto())

	r, err := catalogo.Consultar(context.Background(), service.Filtro{CategoriaID: idPerifericos}, service.Pagina{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if productos.filtro != (service.FiltroProductos{CategoriaID: idPerifericos, SoloVendibles: true}) {
		t.Fatalf("filtro enviado a Productos: %+v", productos.filtro)
	}
	if r.Total != 1 || r.Items[0].ID != idTeclado {
		t.Fatalf("un producto con permite_venta = false llegó al catálogo: %+v", r.Items)
	}
}

func TestConsultarLimitaLlamadasSimultaneasAStock(t *testing.T) {
	stock := &stockLento{}
	catalogo := service.NuevoCatalogo(muchosProductos{}, stock)

	r, err := catalogo.Consultar(context.Background(), service.Filtro{}, service.Pagina{})
	if err != nil || r.Total != 40 {
		t.Fatalf("total=%d err=%v", r.Total, err)
	}
	if stock.maximo > 8 {
		t.Fatalf("hubo %d llamadas simultáneas a Stock, el máximo es 8", stock.maximo)
	}
}

func TestObtener(t *testing.T) {
	stock := stockCompleto()
	stock.stock[idMouse] = 0
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stock)
	ctx := context.Background()

	item, err := catalogo.Obtener(ctx, idTeclado)
	if err != nil || item.ID != idTeclado || item.Stock != 15 {
		t.Fatalf("detalle del teclado: %+v, err=%v", item, err)
	}
	if item.Moneda != "CLP" || len(item.CategoriaIDs) != 1 || item.CategoriaIDs[0] != idPerifericos {
		t.Fatalf("moneda o categorías del teclado: %+v", item)
	}
	for _, id := range []string{idMouse, idNoExiste} {
		if _, err := catalogo.Obtener(ctx, id); !errors.Is(err, service.ErrProductoNoEncontrado) {
			t.Fatalf("%s: se esperaba ErrProductoNoEncontrado, obtenido %v", id, err)
		}
	}
}

func TestObtenerPropagaStockNoDisponible(t *testing.T) {
	falla := fmt.Errorf("%w: conexión rechazada", service.ErrDependenciaNoDisponible)
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, &stockFijo{err: falla})

	if _, err := catalogo.Obtener(context.Background(), idTeclado); !errors.Is(err, service.ErrDependenciaNoDisponible) {
		t.Fatalf("se esperaba ErrDependenciaNoDisponible, obtenido %v", err)
	}
}

func TestCategorias(t *testing.T) {
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, stockCompleto())

	categorias, err := catalogo.Categorias(context.Background())
	if err != nil || len(categorias) != 2 {
		t.Fatalf("categorías=%d err=%v", len(categorias), err)
	}
}
