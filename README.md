# Inscripción Moodle y Plataforma de Exámenes

Este proyecto es una aplicación web completa que consta de dos partes principales: un formulario de inscripción online para OpositaTCAE y una plataforma de exámenes.

## Características

### Formulario de Inscripción (`frontend-inscripcion` y `backend`)

*   **Formulario Completo:** Permite a los usuarios rellenar un formulario de inscripción con sus datos personales, académicos y de pago.
*   **Firma Digital:** Incluye un campo para que los usuarios puedan firmar digitalmente el formulario.
*   **Generación de PDF Optimizada:** El backend (Go) genera eficientemente documentos PDF con toda la información y la firma, optimizado para bajo consumo de memoria.
*   **Confirmación por Email:** El usuario y el administrador reciben una copia del PDF de inscripción por correo electrónico.
*   **Integración con Moodle:** Crea y sincroniza automáticamente cuentas de usuario en Moodle.
*   **Rate Limiting:** Protegido contra abusos mediante límites de petición por IP (Redis).

### Plataforma de Exámenes (`frontend-exam` y `backend`)

*   **Gestión de Exámenes:** Panel de administración completo para crear/editar exámenes, revisar intentos y gestionar resultados.
*   **Importación de Resultados Streaming:** Importación eficiente de ficheros Excel masivos con resultados oficiales, utilizando procesamiento por streams (sin carga completa en RAM).
*   **Sincronización Moodle:** Servicio dedicado para mantener los usuarios sincronizados con el curso de Moodle correspondiente.
*   **Resultados y Percentiles:** Cálculo en tiempo real de puntuaciones y percentiles.
*   **Caché de Alto Rendimiento:** Uso de Redis para cachear preguntas y resultados públicos, reduciendo la carga en base de datos.

## Novedades Recientes (Dic 2025)

### Optimización y Backend
*   **Carga Diferida (Lazy Loading):** Optimización masiva en `/admin/results`. Las respuestas de los estudiantes ahora se cargan solo bajo demanda al editar, reduciendo el tamaño de la respuesta inicial y mejorando la velocidad.
*   **Exportación Excel:** Solucionados problemas en el reporte de fallos y respuestas correctas para un análisis más preciso.
*   **Configuración:** Manejo dinámico del remitente SMTP en los mensajes de error.

### Experiencia de Usuario (Frontend)
*   **Identidad Visual:** Actualización de estilos, destacando los méritos con el color `brand-pink`.
*   **Usabilidad:** Implementación global de `cursor-pointer` en todos los elementos interactivos para una navegación más intuitiva.
*   **Feedback:** Nuevas pantallas de "estado vacío" que comunican claramente cuando no hay resultados o intentos registrados.

## Estructura del Proyecto

```
inscripcion-moodle/
│
├── backend/                 # Nuevo backend en Go
│   ├── cmd/                 # Puntos de entrada (api)
│   ├── internal/            # Código privado de la aplicación
│   │   ├── config/          # Configuración
│   │   ├── controllers/     # Handlers HTTP (Admin, Public, Register)
│   │   ├── models/          # Modelos GORM
│   │   ├── services/        # Lógica de negocio (Excel, PDF, Moodle, Auth)
│   │   └── cache/           # Capa de caché Redis
│   ├── go.mod               # Definición de módulo y dependencias
│   └── ...
│
├── frontend-exam/           # Aplicación de exámenes (Astro + React)
│   ├── src/
│   ├── package.json
│   └── astro.config.mjs
│
├── frontend-inscripcion/    # Formulario de inscripción (Vanilla JS)
│   ├── index.html
│   ├── main.js
│   └── style.css
│
├── .gitignore
├── LICENSE
└── README.md
```

## Requisitos

*   **Backend:**
    *   Go 1.25+
    *   Redis (para caché y rate limiting)
    *   Servidor SMTP (para envío de correos)
*   **Frontend (Exams):**
    *   Node.js y npm
*   **Base de Datos:**
    *   MySQL / MariaDB
    *   Moodle (MySQL) para la integración

## Instalación

1.  **Clona el repositorio:**
    ```sh
    git clone https://github.com/tuusuario/inscripcion-moodle.git
    cd inscripcion-moodle
    ```

2.  **Configura el Backend (Go):**
    *   Navega a la carpeta `backend`: `cd backend`
    *   Crea un archivo `.env` basado en la configuración requerida (ver `config` package).
    *   Instala dependencias:
        ```sh
        go mod download
        ```

3.  **Configura el Frontend de Exámenes:**
    *   Navega a la carpeta `frontend-exam`: `cd ../frontend-exam`
    *   Instala las dependencias: `npm install`

## Ejecución

1.  **Backend:**
    *   Desde la carpeta `backend`:
        ```sh
        go run ./cmd/api
        ```
    *   La API estará disponible en el puerto configurado (ej. `8080`).

2.  **Frontend de Exámenes:**
    *   Desde la carpeta `frontend-exam`:
        ```sh
        npm run dev
        ```
    *   Disponible en `http://localhost:4321`.

3.  **Frontend de Inscripción:**
    *   Sirve el archivo `frontend-inscripcion/index.html` con cualquier servidor web estático.

## Licencia

Este proyecto está bajo la Licencia MIT. Consulta el archivo `LICENSE` para más detalles.
