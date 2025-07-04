# Inscripción Moodle

Este proyecto es una aplicación de inscripción online para OpositaTCAE, que permite a los usuarios rellenar un formulario, firmar digitalmente y recibir un PDF de confirmación por email. Además, crea el usuario en Moodle automáticamente.

## Estructura del proyecto

```
inscripcion-moodle/
│
├── backend/
│   ├── main.py              # API FastAPI principal
│   ├── registerdata.py      # Modelo de datos del formulario
│   ├── pdf_utils.py         # Generación de PDF
│   ├── email_utils.py       # Envío de emails
│   ├── moodle_api.py        # Alta de usuario en Moodle
│   ├── rate_limiter.py      # Limitador de peticiones por IP
│   ├── requirements.txt     # Dependencias Python
│   └── assets/              # Recursos (fuentes, imágenes)
│
├── frontend/
│   ├── index.html           # Formulario web
│   ├── main.js              # Lógica JS del formulario
│   └── style.css            # Estilos personalizados
│
├── .env                     # Variables de entorno (no subir a git)
├── .gitignore
└── README.md
```

## Requisitos

- Python 3.10+
- Node.js (opcional, solo si usas herramientas de desarrollo frontend)
- [pip](https://pip.pypa.io/en/stable/installation/)
- [python-dotenv](https://pypi.org/project/python-dotenv/) (`pip install python-dotenv`)

## Instalación

1. **Clona el repositorio:**
   ```sh
   git clone https://github.com/tuusuario/inscripcion-moodle.git
   cd inscripcion-moodle
   ```

2. **Instala las dependencias del backend:**
   ```sh
   cd backend
   pip install -r requirements.txt
   ```

3. **Configura el archivo `.env` en la carpeta `backend/`:**
   ```
   SMTP_USER=tu_usuario
   SMTP_PASS=tu_contraseña
   SMTP_SERVER=smtp.tuservidor.com
   SMTP_PORT=587
   ADMIN_EMAIL=admin@tudominio.com
   MOODLE_URL=https://tudominio.com/moodle
   MOODLE_TOKEN=tu_token
   ```

4. **(Opcional) Instala dependencias frontend si usas herramientas adicionales.**

## Ejecución

### Backend (API)

Desde la carpeta raíz del proyecto:

```sh
uvicorn backend.main:app --reload
```

- El backend estará disponible en [http://localhost:8000](http://localhost:8000)
- Documentación interactiva en [http://localhost:8000/docs](http://localhost:8000/docs)

### Frontend

Puedes abrir `frontend/index.html` directamente en tu navegador para pruebas locales, o servirlo desde un servidor web (por ejemplo, Apache o Nginx).

**Nota:** Si sirves el frontend desde un dominio diferente, asegúrate de configurar correctamente los orígenes permitidos (`allow_origins`) en `backend/main.py`.

## Personalización

- Modifica los campos del formulario en `frontend/index.html` y el modelo en `backend/registerdata.py` si necesitas más datos.
- Cambia los textos y estilos en `frontend/style.css` y los emails en `backend/email_utils.py`.

## Seguridad

- **No subas tu archivo `.env` ni datos sensibles al repositorio.**
- Configura correctamente CORS en producción para aceptar solo tu dominio.

## Licencia

MIT