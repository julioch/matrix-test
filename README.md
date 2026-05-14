# Matrices QR y Stats

## Arquitectura

- **api-go (Fiber / Go)**: recibe una matriz rectangular, valida y calcula su factorización QR. Luego invoca la API de Node para calcular estadísticas.
- **api-node (Express / Node.js)**: recibe las matrices (Q y R) y calcula: máximo, mínimo, suma, promedio y si alguna matriz es diagonal.
- **Frontend**: Un simple front que obtiene un JWT y llama a la API de Go para mostrar Q/R y estadísticas.

## Requisitos

- Docker Desktop (incluye `docker compose`)
- Opcional para correr sin Docker:
  - Node.js 20+
  - Go 1.22+

## Variables de entorno

Estas variables se usan tanto en local como en Docker:

- `JWT_SECRET`: secreto compartido (HS256).
- `API_KEY`: API key para solicitar JWT en `/token` (header `x-api-key`).
- `NODE_API_URL`: URL de api-node (solo api-go). En Docker se configura a `http://api-node:3000`.
- `CORS_ORIGIN`: orígenes permitidos para el frontend (por defecto incluye `http://localhost:5173` y `http://127.0.0.1:5173`).

## Ejecutar con Docker (recomendado)

### 1) Definir variables para Docker Compose

Docker Compose lee automáticamente un archivo `.env` en la raíz del proyecto.

Crea un archivo `.env` (en la raíz) con:

```env
JWT_SECRET=dev-secret-largo
API_KEY=dev-api-key
```

Opcional:

```env
CORS_ORIGIN=http://localhost:5173,http://127.0.0.1:5173
```

### 2) Levantar servicios

```bash
docker compose up --build -d
```

Puertos:

- api-go: `http://localhost:8080`
- api-node: `http://localhost:3001` (expuesto solo para debug; api-go consume a api-node por red interna Docker)

## Frontend

El frontend está en `frontend/` (Vite + Vue).

Ejecuta:

```bash
cd frontend
npm install
npm run dev
```

Abre:

- `http://127.0.0.1:5173` (o `http://localhost:5173`)

En la UI:

- Ingresa `API_KEY` (la del `.env`)
- Click en **Obtener token**
- Click en **Procesar QR**

## Probar APIs (Insomnia / Postman)

### 1) Health check

- `GET http://localhost:8080/health`
- `GET http://localhost:3001/health`

### 2) Obtener token (JWT)

- `POST http://localhost:8080/token`
- Header: `x-api-key: <API_KEY>`
- Body:

```json
{ "sub": "test" }
```

### 3) Calcular QR (protegido con JWT)

- `POST http://localhost:8080/qr`
- Header: `Authorization: Bearer <token>`
- Body (ejemplo m>=n):

```json
{ "matrix": [[1,2],[3,4],[5,6]] }
```

## Tests unitarios

### Node

```bash
cd api-node
npm install
npm test
```

### Go

```bash
cd api-go
go test ./...
```
