/healthz: 200
/readyz: 200
/api/estado: 200
/api/docs: 200
/api/catalogo: 200, 3 productos vendibles
/api/catalogo?categoria=Perif\u00e9ricos: 200, 2 productos
/api/catalogo?categoria=Inexistente: 200, lista vac\u00eda
/api/catalogo/{id} existente: 200
/api/catalogo/{id} inexistente: 404 problem+json
/api/catalogo/{id} no UUID: 422 problem+json
/api/categorias: 200, 2 categor\u00edas
prod /api/docs: 404
prod /api/openapi.yaml: 404
Configuraci\u00f3n incompleta (sin DATABASE_URL): c\u00f3digo 1
PostgreSQL no disponible: 503
