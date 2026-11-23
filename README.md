# Training & Judge Center

Plataforma para la formación en programación competitiva: permite a coaches y estudiantes organizar **grupos**, publicar **problemas** y **materiales**, crear **concursos** con marcador en vivo, y enviar **soluciones** que se evalúan automáticamente en un sandbox aislado.

Este repositorio es el **centro de documentación y especificación** del proyecto. El código del backend vive en [`training-and-judge-center-backend/`](./training-and-judge-center-backend/README.md).

---

## Tabla de contenidos

- [¿Qué problema resuelve?](#qué-problema-resuelve)
- [Funcionalidades](#funcionalidades)
- [Arquitectura general](#arquitectura-general)
- [Estructura del repositorio](#estructura-del-repositorio)
- [Especificaciones funcionales](#especificaciones-funcionales)
- [Configuración de la plataforma](#configuración-de-la-plataforma)
- [Inicio rápido](#inicio-rápido)
- [Documentación técnica](#documentación-técnica)
- [Flujo de contribución](#flujo-de-contribución)

---

## ¿Qué problema resuelve?

Los entrenamientos de programación competitiva dependen de tres cosas: tener buenos problemas, competir contra otros en condiciones controladas y recibir retroalimentación inmediata sobre el código. Esta plataforma integra las tres en un solo sistema:

- **Gestión académica**: coaches crean grupos, invitan estudiantes y aprueban solicitudes de ingreso.
- **Banco de problemas**: problemas con enunciado, límites de tiempo y memoria, casos de prueba, y checkers/validators personalizados.
- **Concursos**: competencias con inscripción, equipos, marcador (standings) y resultados que se revelan en un momento definido.
- **Juez automático**: cada envío se compila y ejecuta dentro de contenedores aislados, sin red y con sistema de archivos de solo lectura.

## Funcionalidades

| Dominio | Qué permite |
|---------|-------------|
| **Usuarios** | Registro, login con JWT, perfil y dashboard personal. |
| **Grupos** | Crear, editar y eliminar grupos; unirse por solicitud o por invitación; administrar miembros y solicitudes. |
| **Problemas** | Crear, editar, eliminar y publicar problemas; importación desde archivo; control de visibilidad; estadísticas; rejuzgamiento de envíos. |
| **Materiales** | Crear, editar y buscar materiales de estudio con control de visibilidad. |
| **Concursos** | Crear, editar y eliminar concursos; inscripción; consulta de envíos y standings; revelación de resultados. |
| **Equipos** | Equipos de participantes dentro de concursos. |
| **Envíos** | Enviar código, consultar su estado y veredicto, descargar el código fuente y cambiar su visibilidad. |
| **Juez** | Pools de contenedores sandbox (`heavy` y `light`), compilación, ejecución, checkers y validators, y un worker que consume la cola de envíos. |

Las especificaciones detalladas de cada caso de uso están en [`specs/`](./specs/).

## Arquitectura general

```
┌──────────────┐     ┌────────────────────┐     ┌──────────────────┐
│   Frontend   │────▶│  Backend API (Go)  │────▶│  PostgreSQL 16   │
│ (repo aparte)│     │  DDD + Hexagonal   │     │  (datos, estado) │
└──────────────┘     └─────────┬──────────┘     └──────────────────┘
                               │
                 ┌─────────────┼──────────────┐
                 ▼             ▼              ▼
          ┌────────────┐ ┌──────────┐ ┌─────────────────┐
          │ RabbitMQ   │ │ MongoDB  │ │ Almacenamiento  │
          │ (cola de   │ │ (marcador│ │ local o GCS     │
          │  envíos)   │ │  de      │ │ (problemas,     │
          └─────┬──────┘ │ concursos│ │  código fuente) │
                │        └──────────┘ └─────────────────┘
                ▼
        ┌────────────────────┐
        │ Worker (juez)      │
        │ pools de sandbox   │
        │ heavy / light      │
        └────────────────────┘
```

- **Backend API**: expone la API REST. Organizado con **DDD + Arquitectura Hexagonal**; la dependencia siempre apunta hacia el dominio (`domain/` ← `application/` ← `adapter/` ← `cmd/`).
- **Worker (juez)**: consume la cola de envíos, ejecuta cada solución en un contenedor sandbox y registra el veredicto.
- **Almacenamiento**: `local` en desarrollo, Google Cloud Storage en producción.
- **Despliegue**: Docker Compose para desarrollo local y Google Kubernetes Engine (GKE) para staging/producción.

El frontend es un proyecto independiente (React + Node.js) que consume la API.

## Estructura del repositorio

```
training-center-specs/
├── README.md                          ← este documento
├── CLAUDE.md                          ← guía de navegación para asistentes de código
├── plan.md                            ← plan de implementación
├── specs/                             ← especificaciones funcionales por dominio
│   ├── Platform configuration.md      ← límites globales y configuración estática
│   ├── Contest management/
│   ├── Group management/
│   ├── Judge System/
│   ├── Material management/
│   ├── Problem management/
│   ├── Submission management/
│   ├── Team management/
│   ├── Users management/
│   └── templates/                     ← plantillas para nuevas especificaciones
├── model/
│   └── diagram.md                     ← modelo de datos
├── training-and-judge-center-backend/ ← código del backend y del juez (Go)
│   ├── cmd/                           ← binarios: api, worker, migrate, seed, ...
│   ├── internal/                      ← domain, application, adapter, config
│   ├── tests/                         ← pruebas end-to-end
│   ├── ARCHITECTURE.md                ← fuente de verdad de la arquitectura
│   └── JUDGE_SANDBOX_REDESIGN.md      ← diseño vigente del juez
└── .github/
    ├── PULL_REQUEST_TEMPLATE.md
    └── workflows/                     ← CI, cobertura y despliegue a GKE
```

## Especificaciones funcionales

Cada dominio tiene una carpeta en `specs/` con un `README.md` de contexto y una subcarpeta por caso de uso con su `spec.md`.

| Dominio | Casos de uso |
|---------|--------------|
| [Problem management](./specs/Problem%20management/README.md) | Crear, ver, actualizar, eliminar, cambiar visibilidad, estadísticas, rejuzgar envíos, ver archivos. |
| [Group management](./specs/Group%20management/README.md) | Crear, ver, actualizar, eliminar grupos; unirse; invitar; gestionar miembros y solicitudes. |
| [Contest management](./specs/Contest%20management/README.md) | Crear, ver, actualizar, eliminar concursos; inscribirse; ver standings y envíos; revelar resultados. |
| [Material management](./specs/Material%20management/README.md) | Crear, ver, actualizar, eliminar, buscar y cambiar visibilidad de materiales. |
| [Submission management](./specs/Submission%20management) | Ver, descargar y cambiar la visibilidad de envíos. |
| [Judge System](./specs/Judge%20System/README.md) | Envío a juez y gestión del pool de contenedores. |
| [Team management](./specs/Team%20management) | Equipos dentro de concursos. |
| [Users management](./specs/Users%20management) | Registro, autenticación y perfil. |

Las plantillas para nuevas especificaciones están en [`specs/templates/`](./specs/templates/).

## Configuración de la plataforma

La plataforma tiene dos grupos de parámetros **estáticos** (requieren un despliegue para cambiar, por seguridad):

- **Límites globales** de tiempo y memoria para problemas, con excepciones por lenguaje (`maxTimeLimitGlobal`, `maxMemoryLimitGlobal`, `languageOverrides`).
- **Configuración del sandbox**: directorio de ejecución, tamaño de tmpfs, límite de procesos, red deshabilitada y sistema de archivos de solo lectura.

El detalle completo, con ejemplos y las reglas de validación, está en [**Platform configuration.md**](./specs/Platform%20configuration.md).

Lenguajes soportados actualmente: `cpp17`, `cpp20`, `java17` y `python310`.

## Inicio rápido

### Requisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) en ejecución.
- [Go 1.25+](https://go.dev/dl/) si se quiere ejecutar el backend fuera de Docker.

### Levantar el backend con Docker Compose

```bash
cd training-and-judge-center-backend
docker-compose --env-file .env up --build -d
```

Esto levanta PostgreSQL, ejecuta las migraciones y arranca la API. Las instrucciones detalladas (autenticación, ejecución local, variables de entorno) están en el [README del backend](./training-and-judge-center-backend/README.md).

### Comandos frecuentes (desde `training-and-judge-center-backend/`)

| Acción | Comando |
|--------|---------|
| Ejecutar la API | `go run cmd/api/main.go` |
| Migraciones | `go run cmd/migrate/main.go up\|down\|status` |
| Crear el administrador inicial | `go run cmd/seed/main.go` (idempotente, usa `ADMIN_*`) |
| Ejecutar el worker del juez | `go run cmd/worker/main.go` |
| Pruebas unitarias e integración | `go test ./...` |
| Pruebas end-to-end | `bash tests/test_e2e.sh` (requiere la API en ejecución) |

En desarrollo local, `STORAGE_BACKEND=local` usa el sistema de archivos en lugar de GCS.

## Documentación técnica

| Documento | Para qué sirve |
|-----------|----------------|
| [ARCHITECTURE.md](./training-and-judge-center-backend/ARCHITECTURE.md) | Convenciones de capas, nombrado y ubicación de tipos. **Fuente de verdad.** |
| [JUDGE_SANDBOX_REDESIGN.md](./training-and-judge-center-backend/JUDGE_SANDBOX_REDESIGN.md) | Diseño vigente del juez y su sandbox. |
| [RUNNER_ARCHITECTURE.md](./training-and-judge-center-backend/RUNNER_ARCHITECTURE.md) | Diseño **anterior** del runner (histórico; ver nota abajo). |
| [GKE_DEPLOY_ROADMAP.md](./training-and-judge-center-backend/GKE_DEPLOY_ROADMAP.md) | Hoja de ruta del despliegue en GKE. |
| [Training & Judge Center API.yml](./training-and-judge-center-backend/Training%20%26%20Judge%20Center%20API.yml) | Especificación OpenAPI de la API. |
| [model/diagram.md](./model/diagram.md) | Modelo de datos. |
| [plan.md](./plan.md) | Plan de implementación original. |

> **Nota:** `RUNNER_ARCHITECTURE.md` describe el diseño previo al rediseño del sandbox. Para el comportamiento actual del juez, consultar `JUDGE_SANDBOX_REDESIGN.md`.

## Flujo de contribución

1. Crear la rama desde `develop` actualizado.
2. Seguir las convenciones de [ARCHITECTURE.md](./training-and-judge-center-backend/ARCHITECTURE.md) y, si se cambia un comportamiento, actualizar la especificación correspondiente en `specs/`.
3. Asegurar que `go build ./...` y `go test ./...` pasen en `training-and-judge-center-backend/`.
4. Abrir el PR contra **`develop`** (nunca contra `main`) usando la plantilla de `.github/PULL_REQUEST_TEMPLATE.md`.

El CI se ejecuta en cada PR hacia `develop` y `main`: verifica el BOM en los fuentes Go, las pruebas con PostgreSQL y la cobertura.
