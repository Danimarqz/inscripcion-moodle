# Inscripción Moodle y Plataforma de Exámenes

Aplicación web para OpositaTCAE compuesta por **tres desplegables** que comparten un único backend:

- **`backend/`** — API en Go (chi, GORM, MariaDB/MySQL, Redis). Sirve tanto el flujo de inscripción como la plataforma de exámenes.
- **`frontend-exam/`** — Plataforma de exámenes en Astro 6 + Preact + Tailwind v4 (panel de administración + examen público). Puerto `4321`.
- **`frontend-inscripcion/`** — Formulario de inscripción estático (HTML/JS vanilla, servido con nginx).

## Características

### Inscripción (`frontend-inscripcion` + `backend`)

- **Formulario completo** con datos personales, académicos y firma digital.
- **Generación de PDF** con `gofpdf`, optimizada para bajo consumo de memoria.
- **Confirmación por email** (usuario y administrador) con el PDF adjunto.
- **Integración con Moodle:** alta y sincronización automática de cuentas.
- **Rate limiting** por IP respaldado en Redis.

### Plataforma de exámenes (`frontend-exam` + `backend`)

- **Gestión de exámenes:** panel admin para crear/editar exámenes y preguntas, revisar intentos y gestionar resultados.
- **Modos de puntuación:** `legacy` (penalización configurable por fallo) y `absolute` (puntos por acierto/fallo), con nota sobre bases secundarias.
- **Criterios de aprobado, pesos y méritos:** umbral de corte, nota ponderada (peso examen + méritos) y ranking de méritos.
- **Percentiles sobre grupos de exámenes:** un examen puede asociarse con otros (relación recíproca) para calcular el percentil sobre el conjunto. El recálculo de percentiles es asíncrono y *coalescido* por examen.
- **Resultados oficiales:**
  - Importación de Excel masivo por *streaming* (sin cargar el fichero en RAM).
  - Conversión PDF→Excel mediante Lambda integrada en el panel.
  - Vinculación a usuarios por DNI/nombre con sincronización Moodle y feedback en tiempo real vía **SSE**.
  - Las notas oficiales se aplican automáticamente (`COALESCE(oficial, simulador)`) cuando existen, sin flag manual.
- **Carga diferida:** las preguntas y las respuestas de cada intento se cargan bajo demanda; los listados omiten campos pesados (`answers_json`) y asociaciones no usadas.
- **Caché Redis** para listado/exámenes por slug, preguntas y resultados públicos; throttling de logins de admin fallidos.

## Arquitectura del backend

Punto de entrada: `backend/cmd/api/main.go` → `internal/server/server.go` (cablea todas las dependencias). Capas bajo `internal/`:

- `config/` — carga de entorno (`backend/.env`).
- `storage/` — clientes MariaDB (GORM) y Redis. **AutoMigrate desactivado**; el esquema se cambia con SQL manual en `backend/db/migrations/`.
- `repository/` — acceso a datos GORM.
- `services/` — lógica de negocio por dominio: `auth`, `email`, `exam`, `excelimport`, `moodle`, `admin`, `pdf`. Email y Moodle usan *worker pools* inicializados una vez.
- `controllers/` — handlers HTTP agrupados (`PublicController`, `RegisterController`, `AdminController`).
- `middleware/` — middleware chi + `RateLimiter` respaldado en Redis.
- `cache/` — capa de caché Redis.

## Estructura del proyecto

```
inscripcion-moodle/
├── backend/                 # API en Go
│   ├── cmd/api/             # Punto de entrada
│   ├── internal/            # config, controllers, services, repository, models, cache, middleware
│   ├── db/migrations/       # Migraciones SQL manuales
│   └── go.mod
├── frontend-exam/           # Plataforma de exámenes (Astro 6 + Preact + Tailwind v4)
│   ├── src/                 # pages, components, services, hooks, utils, types
│   └── astro.config.mjs
├── frontend-inscripcion/    # Formulario de inscripción (HTML/JS vanilla + nginx.conf)
├── docker-compose.yml       # Despliegue con imágenes prebuilt de GHCR
└── README.md
```

## Requisitos

- **Backend:** Go 1.26+, Redis, servidor SMTP.
- **Frontend exámenes:** Node.js + npm.
- **Base de datos:** MariaDB/MySQL (y Moodle para la integración).

## Desarrollo

### Backend (desde `backend/`)

```sh
go run ./cmd/api          # arranca la API (lee backend/.env)
go build ./cmd/api        # compila
go test ./...             # tests
golangci-lint run ./...   # lint (CI usa v2.6)
```

> `backend/.env` está gitignored y **no** hay plantilla committeada (decisión intencionada).

### Frontend de exámenes (desde `frontend-exam/`)

```sh
npm install
npm run dev      # astro dev en :4321
npm run build
npm run preview
```

### Frontend de inscripción

Servir `frontend-inscripcion/index.html` con cualquier servidor estático (config nginx en `frontend-inscripcion/nginx.conf`).

### Atajo local

`make dev` levanta backend (`:8080`) y frontend-exam (`:4321`) a la vez.

## Despliegue

Producción vía `docker-compose.yml` con imágenes prebuilt en GHCR (`ghcr.io/danimarqz/backend-moodle`, `ghcr.io/danimarqz/frontend-exam`). Espera una red externa `nginx_proxy` y lee `backend/.env`. Las migraciones SQL de `backend/db/migrations/` se aplican manualmente.

## Licencia

Licencia MIT. Ver `LICENSE`.
