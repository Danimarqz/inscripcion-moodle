# Despliegue y tareas comunes del proyecto.
#
# Configuracion del servidor en deploy.env (gitignoreado). Crea uno a partir
# de deploy.env.example:
#
#   cp deploy.env.example deploy.env
#
# Uso:
#   make dev             # backend + frontend en local (paralelo)
#   make deploy          # backend + frontend
#   make backend         # solo backend
#   make frontend        # solo frontend
#   make build-backend   # compila el binario sin subirlo
#   make build-frontend  # build de Astro sin subirlo
#   make test            # tests del backend Go
#   make test-frontend   # tests de vitest
#   make test-all        # ambos
#
# deploy/backend/frontend corren sus tests antes de compilar y subir.

SHELL := /bin/bash

# make usa un bash no interactivo, que no carga nvm desde ~/.zshrc / ~/.bashrc.
# Sin esto, en WSL `npm` resuelve al npm de Windows en /mnt/c y la instalacion
# revienta con EISDIR/EPERM. NVM_USE carga nvm en el target que lo necesite.
NVM_USE := export NVM_DIR="$$HOME/.nvm"; [ -s "$$NVM_DIR/nvm.sh" ] && . "$$NVM_DIR/nvm.sh" >/dev/null;

# Carga deploy.env si existe. Las variables quedan disponibles para los targets
# y tambien se exportan a los subprocesos (scp/ssh).
-include deploy.env
export

REMOTE := $(RemoteUser)@$(RemoteHost)

.PHONY: dev dev-backend dev-frontend deploy backend frontend build-backend build-frontend \
        upload-backend upload-frontend test test-frontend test-all check-deploy-env

dev:
	@echo "==> Arrancando backend (:8080) y frontend-exam (:4321). Ctrl+C para parar."
	@trap 'kill 0' INT TERM EXIT; \
		$(MAKE) --no-print-directory dev-backend & \
		$(MAKE) --no-print-directory dev-frontend & \
		wait

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	$(NVM_USE) cd frontend-exam && npm run dev

deploy: backend frontend

# Los tests corren antes de compilar: si fallan, no se sube nada al servidor.
backend: check-deploy-env test build-backend upload-backend
	@echo "==> Ejecutando update-backend.sh en el servidor"
	ssh $(REMOTE) './update-backend.sh'

frontend: check-deploy-env test-frontend build-frontend upload-frontend
	@echo "==> Ejecutando update-astro.sh en el servidor"
	ssh $(REMOTE) './update-astro.sh'

build-backend:
	@echo "==> Compilando backend Go (linux/amd64)"
	cd backend && GOOS=linux GOARCH=amd64 go build -o opositatcae-api ./cmd/api

PROD_API_URL := https://simulador.opositatcae.es/api

build-frontend:
	@echo "==> Build Astro (PUBLIC_API_URL=$(PROD_API_URL))"
	$(NVM_USE) cd frontend-exam && PUBLIC_API_URL='$(PROD_API_URL)' npm run build
	@echo "==> Empaquetando dist en dist.tar"
	cd frontend-exam && tar -cf dist.tar dist

upload-backend: check-deploy-env
	@echo "==> Subiendo binario a $(REMOTE):$(BackendDest)"
	scp backend/opositatcae-api '$(REMOTE):$(BackendDest)'

upload-frontend: check-deploy-env
	@echo "==> Subiendo dist.tar a $(REMOTE):$(FrontendDest)"
	scp frontend-exam/dist.tar '$(REMOTE):$(FrontendDest)'

test:
	@echo "==> Tests backend (Go)"
	cd backend && go test ./...

test-frontend:
	@echo "==> Tests frontend (vitest)"
	$(NVM_USE) cd frontend-exam && npm test

test-all: test test-frontend

check-deploy-env:
	@if [[ -z "$(RemoteUser)" || -z "$(RemoteHost)" || -z "$(BackendDest)" || -z "$(FrontendDest)" ]]; then \
		echo "Falta configuracion en deploy.env. Copialo desde deploy.env.example." >&2; \
		exit 1; \
	fi
