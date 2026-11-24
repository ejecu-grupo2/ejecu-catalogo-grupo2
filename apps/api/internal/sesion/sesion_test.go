package sesion_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/sesion"
)

func TestTokenDe(t *testing.T) {
	casos := []struct {
		nombre, authorization, cookie, esperado string
	}{
		{"bearer entre servicios", "Bearer abc.def.ghi", "", "abc.def.ghi"},
		{"cookie del navegador", "", "otra=1; sesion=jwt-cookie", "jwt-cookie"},
		{"bearer tiene prioridad", "Bearer desde-header", "sesion=desde-cookie", "desde-header"},
		{"sin credenciales", "", "", ""},
		{"cookie sin sesion", "", "tema=oscuro", ""},
		{"authorization que no es bearer", "Basic dXN1YXJpbw==", "", ""},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenido := sesion.TokenDe(caso.authorization, caso.cookie); obtenido != caso.esperado {
				t.Fatalf("se esperaba %q, obtenido %q", caso.esperado, obtenido)
			}
		})
	}
}

func TestCredencialesEnContexto(t *testing.T) {
	if c := sesion.De(context.Background()); c != (sesion.Credenciales{}) {
		t.Fatalf("un contexto sin credenciales debe devolver valores vacíos, obtenido %+v", c)
	}
	ctx := sesion.Con(context.Background(), sesion.Credenciales{Token: "t", RequestID: "r"})
	if c := sesion.De(ctx); c.Token != "t" || c.RequestID != "r" {
		t.Fatalf("credenciales recuperadas: %+v", c)
	}
}

func TestNuevoRequestID(t *testing.T) {
	a, b := sesion.NuevoRequestID(), sesion.NuevoRequestID()
	if len(a) != 32 || a == b {
		t.Fatalf("ids generados: %q y %q", a, b)
	}
}

func TestReenviar(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://stock.invalid", nil)
	if err := sesion.Reenviar(context.Background(), req); err != nil || len(req.Header) != 0 {
		t.Fatalf("sin credenciales no debe agregar encabezados: %v, err=%v", req.Header, err)
	}
	ctx := sesion.Con(context.Background(), sesion.Credenciales{Token: "jwt", RequestID: "req-1"})
	if err := sesion.Reenviar(ctx, req); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer jwt" || req.Header.Get("X-Request-Id") != "req-1" {
		t.Fatalf("encabezados reenviados: %v", req.Header)
	}
}
