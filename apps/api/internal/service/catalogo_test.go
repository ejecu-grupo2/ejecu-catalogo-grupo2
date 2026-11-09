package service_test

import (
	"context"
	"testing"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
)

func TestConsultarCatalogo(t *testing.T) {
	ctx := context.Background()

	t.Run("sin filtro retorna todos los items", func(t *testing.T) {
		items, err := service.ConsultarCatalogo(ctx, "")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(items) == 0 {
			t.Fatal("se esperaba lista con items")
		}
	})

	t.Run("filtro existente retorna coincidencias", func(t *testing.T) {
		items, err := service.ConsultarCatalogo(ctx, "Periféricos")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(items) == 0 {
			t.Fatal("se esperaban productos")
		}
	})

	t.Run("filtro inexistente retorna lista vacia", func(t *testing.T) {
		items, err := service.ConsultarCatalogo(ctx, "Inexistente")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("se esperaba lista vacía, obtenido %d", len(items))
		}
	})
}

func TestObtenerItemCatalogo(t *testing.T) {
	ctx := context.Background()

	t.Run("producto existente", func(t *testing.T) {
		item, err := service.ObtenerItemCatalogo(ctx, "prod-101")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if item.ID != "prod-101" {
			t.Errorf("id incorrecto: %s", item.ID)
		}
	})

	t.Run("producto inexistente", func(t *testing.T) {
		_, err := service.ObtenerItemCatalogo(ctx, "inexistente")
		if err != service.ErrProductoNoEncontrado {
			t.Fatalf("esperado ErrProductoNoEncontrado, obtenido: %v", err)
		}
	})
}

func TestConsultarCategorias(t *testing.T) {
	ctx := context.Background()
	cats, err := service.ConsultarCategorias(ctx)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cats) == 0 {
		t.Fatal("se esperaban categorias")
	}
}
