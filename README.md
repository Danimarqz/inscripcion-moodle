# Inscripción Moodle y Plataforma de Exámenes

Este proyecto es una aplicación web completa que consta de dos partes principales: un formulario de inscripción online para OpositaTCAE y una plataforma de exámenes.

## Características

### Formulario de Inscripción (`frontend-inscripcion` y `backend`)

*   **Formulario Completo:** Permite a los usuarios rellenar un formulario de inscripción con sus datos personales, académicos y de pago.
*   **Firma Digital:** Incluye un campo para que los usuarios puedan firmar digitalmente el formulario.
*   **Generación de PDF:** Una vez enviado el formulario, el backend genera un documento PDF con toda la información y la firma.
*   **Confirmación por Email:** El usuario y el administrador reciben una copia del PDF de inscripción por correo electrónico.
*   **Integración con Moodle:** Crea automáticamente una cuenta de usuario en Moodle con los datos del formulario.
*   **Rate Limiting:** Limita el número de peticiones por IP para prevenir abusos.

### Plataforma de Exámenes (`frontend-exam` y `backend`)

*   **Gestión de Exámenes:** El panel de administración se divide en `/admin/exams`, `/admin/submissions` y `/admin/results` para crear/editar exámenes, revisar intentos y gestionar resultados oficiales importados.
*   **Importación de resultados oficiales:** Desde `/admin/results` se puede subir un PDF (procesado con Tabula) y vincular automáticamente los registros con usuarios existentes.
*   **Realización de Exámenes:** Los usuarios pueden acceder a los exámenes activos, responder las preguntas y enviar sus respuestas.
*   **Resultados y Percentiles:** Al finalizar un examen, el usuario recibe su puntuación, percentil y/o detalle de aciertos según lo definido en los flags `show_score`, `show_percentile` y `show_score_full`. La ficha del examen refleja siempre la combinación activa para evitar pasos adicionales al alumno.
*   **Autenticación de Administrador:** El panel de administración está protegido y requiere autenticación mediante JWT.

## Estructura del Proyecto

```
inscripcion-moodle/
│
├── backend/
│   ├── main.py              # API FastAPI principal
│   ├── models/              # Modelos de datos Pydantic
│   ├── db/                  # Configuración de la base de datos y modelos SQLAlchemy
│   ├── routers/             # Rutas de la API (públicas, admin, inscripción)
│   ├── services/            # Lógica de negocio (autenticación, emails, PDF, Moodle)
│   ├── requirements.txt     # Dependencias de Python
│   └── ...
│
├── frontend-exam/
│   ├── src/
│   │   ├── components/      # Componentes Preact para la interfaz de exámenes
│   │   ├── pages/           # Páginas de la aplicación de exámenes (Astro)
│   │   └── services/        # Servicios para conectar con el backend
│   ├── package.json         # Dependencias de Node.js
│   └── astro.config.mjs     # Configuración de Astro
│
├── frontend-inscripcion/
│   ├── index.html           # Formulario de inscripción
│   ├── main.js              # Lógica del formulario (validación, firma, envío)
│   └── style.css            # Estilos del formulario
│
├── .gitignore
├── LICENSE
└── README.md
```

## Requisitos

*   **Backend:**
    *   Python 3.10+
    *   Dependencias en `backend/requirements.txt`
*   **Frontend (Exams):**
    *   Node.js y npm (o un gestor de paquetes compatible)
    *   Dependencias en `frontend-exam/package.json` (incluye `@astrojs/node` como adaptador de servidor)
*   **Base de Datos:**
    *   Un servidor de base de datos compatible con SQLAlchemy (ej. MySQL, PostgreSQL).
    *   Ejecuta las migraciones SQL en `backend/db/migrations/` (por ejemplo `20250926_exam_user_and_official_results.sql` y `20250927_exam_official_result_nullable_user.sql`) antes de iniciar el nuevo backend.

## Instalación

1.  **Clona el repositorio:**
    ```sh
    git clone https://github.com/tuusuario/inscripcion-moodle.git
    cd inscripcion-moodle
    ```

2.  **Configura el Backend:**
    *   Navega a la carpeta `backend`: `cd backend`
    *   Crea un entorno virtual: `python -m venv venv`
    *   Activa el entorno virtual:
        *   Windows: `venv\Scripts\activate`
        *   macOS/Linux: `source venv/bin/activate`
    *   Instala las dependencias: `pip install -r requirements.txt`
    *   Crea un archivo `.env` en la raíz de la carpeta `backend` y configúralo con tus variables de entorno. Puedes usar `backend/.env.example` como plantilla.

3.  **Configura el Frontend de Exámenes:**
    *   Navega a la carpeta `frontend-exam`: `cd ../frontend-exam`
    *   Instala las dependencias: `npm install`
    *   Crea un archivo `.env` en la raíz de la carpeta `frontend-exam` y define la variable `PUBLIC_API_URL` con la URL de tu backend (ej. `PUBLIC_API_URL=http://localhost:8000`).

## Ejecución

1.  **Backend:**
    *   Desde la carpeta `backend`, con el entorno virtual activado, ejecuta en desarrollo:
        ```sh
        uvicorn main:app --reload
        ```
    *   En servidores con pocos recursos define las variables y flags recomendados antes de exponer el servicio:
        ```sh
        PYTHONMALLOC=malloc UVICORN_WORKERS=1 uvicorn main:app \
          --host 0.0.0.0 --port 8000 \
          --limit-concurrency 20 \
          --timeout-keep-alive 5
        ```
    *   La API estará disponible en `http://localhost:8000`.

2.  **Frontend de Exámenes:**
    *   Desde la carpeta `frontend-exam`, ejecuta:
        ```sh
        npm run dev
        ```
    *   La aplicación de exámenes estará disponible en `http://localhost:4321`.
    *   Para entornos de producción ejecuta `npm run build`; Astro está configurado en modo `server` con el adaptador Node.

3.  **Frontend de Inscripción:**
    *   Abre el archivo `frontend-inscripcion/index.html` directamente en tu navegador o sírvelo con un servidor web.
    *   **Importante:** Para que el formulario de inscripción pueda comunicarse con el backend, necesitarás servirlo desde un dominio o configurar CORS adecuadamente en `backend/main.py` para permitir el origen desde el que se sirve el archivo.

## Licencia

Este proyecto está bajo la Licencia MIT. Consulta el archivo `LICENSE` para más detalles.

## Notas recientes

*   La importación de resultados oficiales utiliza `tabula-py` con `guess=false`; los registros que no se puedan asociar a un candidato existente se guardan con `user_id` vacío.
*   La página de examen (`/exam/[id]`) recupera la configuración del examen en cada petición para respetar cambios en `show_score`, `show_percentile` y `show_score_full` sin recompilar.
*   Los mensajes de envío y de reintentos se muestran integrados en la interfaz en lugar de usar `alert()` del navegador.
