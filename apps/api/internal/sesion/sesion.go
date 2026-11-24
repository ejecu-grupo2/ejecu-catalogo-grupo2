package sesion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// NombreCookie es la cookie donde el servicio de identidad entrega el JWT.
// Debe coincidir con el contrato del servicio de identidad.
const NombreCookie = "sesion"

// Credenciales son los datos del usuario que se reenvían a otros servicios.
type Credenciales struct {
	Token     string
	RequestID string
}

type clave struct{}

// Con devuelve un contexto que lleva las credenciales.
func Con(ctx context.Context, c Credenciales) context.Context {
	return context.WithValue(ctx, clave{}, c)
}

// De recupera las credenciales del contexto. Si no hay, devuelve valores vacíos.
func De(ctx context.Context) Credenciales {
	c, _ := ctx.Value(clave{}).(Credenciales)
	return c
}

// Reenviar agrega a una request hacia otro servicio el JWT (como Bearer) y el
// X-Request-Id guardados en ctx, como pide el DAS. Tiene la firma de
// RequestEditorFn de los clientes de oapi-codegen, así que se pasa directo.
func Reenviar(ctx context.Context, req *http.Request) error {
	c := De(ctx)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.RequestID != "" {
		req.Header.Set("X-Request-Id", c.RequestID)
	}
	return nil
}

// TokenDe obtiene el JWT desde el encabezado Authorization (llamadas entre servicios)
// o, si no viene, desde la cookie de sesión (llamadas desde el navegador).
func TokenDe(authorization, cookie string) string {
	if token, ok := strings.CutPrefix(authorization, "Bearer "); ok {
		return strings.TrimSpace(token)
	}
	if cookie == "" {
		return ""
	}
	cookies, err := http.ParseCookie(cookie)
	if err != nil {
		return ""
	}
	for _, c := range cookies {
		if c.Name == NombreCookie {
			return c.Value
		}
	}
	return ""
}

// NuevoRequestID genera un identificador de correlación cuando la request no trae uno.
func NuevoRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
