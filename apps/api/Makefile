generate:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate

# Clientes de servicios externos, desde contracts/externos/.
clientes:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2 -config oapi-codegen/stock.yaml ../../contracts/externos/stock.yaml
