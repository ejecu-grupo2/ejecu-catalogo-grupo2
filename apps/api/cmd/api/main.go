package main

import (
	"context"
	"fmt"
	api "github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/api"
	clientestock "github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/clientes/stock"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/service"
	"github.com/JohannySimken/ejecu-catalogo-grupo2/apps/api/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		documento, err := api.Registrar(http.NewServeMux(), "dev", nil, nil).OpenAPI().YAML()
		if err != nil {
			panic(err)
		}
		fmt.Print(string(documento))
		return
	}
	for _, nombre := range []string{"API_PORT", "API_ENV", "DATABASE_URL", "AUTH_JWKS_URL", "FRONTEND_URL", "STOCK_URL", "TZ"} {
		if os.Getenv(nombre) == "" {
			slog.Error("Falta configuración", "variable", nombre)
			os.Exit(1)
		}
	}
	entorno := os.Getenv("API_ENV")
	if entorno != "dev" && entorno != "test" && entorno != "prod" {
		slog.Error("API_ENV inválido")
		os.Exit(1)
	}
	if os.Getenv("API_PORT") != "8080" {
		slog.Error("API_PORT debe ser 8080")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("Configuración PostgreSQL inválida")
		os.Exit(1)
	}
	defer pool.Close()
	// Timeout explícito de 5 s hacia Stock (sección 2 del DAS).
	fuenteStock, err := clientestock.NuevaFuente(os.Getenv("STOCK_URL"), 5*time.Second)
	if err != nil {
		slog.Error("STOCK_URL inválida")
		os.Exit(1)
	}
	// Productos sigue simulado hasta tener el contrato del grupo 3.
	catalogo := service.NuevoCatalogo(service.ProductosSimulados{}, fuenteStock)
	mux := http.NewServeMux()
	api.Registrar(mux, entorno, store.Postgres{Pool: pool}, catalogo)
	servidor := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		limite, fin := context.WithTimeout(context.Background(), 5*time.Second)
		defer fin()
		_ = servidor.Shutdown(limite)
	}()
	slog.Info("Iniciando servicio", "servicio", "catalogo", "version", version)
	if err := servidor.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Fallo del servidor")
		os.Exit(1)
	}
}
