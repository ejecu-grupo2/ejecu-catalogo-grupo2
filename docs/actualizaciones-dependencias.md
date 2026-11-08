# Actualizaciones de dependencias

| Dependencia                    | Version DAS original | Version corregida | Motivo                                                                                      |
| ------------------------------ | -------------------- | ----------------- | ------------------------------------------------------------------------------------------- |
| Vite                           | 8.1.5                | 8.1.6             | Corregir CVE-2026-112942 / GHSA-587h-7wmj-h5pm, detectada como HIGH por pnpm audit.         |
| Go toolchain e imagen de build | 1.26.5               | 1.26.10           | Corregir vulnerabilidades alcanzables de la biblioteca estandar detectadas por govulncheck. |
| golang.org/x/text              | v0.29.0              | v0.41.0           | Corregir GO-2026-6629 y GO-2026-5970 detectadas en la conexion PostgreSQL.                  |

Actualizacion aislada en un PR de dependencias. Se conserva el documento DAS
original como referencia y se fija la nueva version exacta en el catalogo y lockfile.
No se omite la auditoria ni se reduce su umbral.

Referencia: https://github.com/advisories/GHSA-587h-7wmj-h5pm
