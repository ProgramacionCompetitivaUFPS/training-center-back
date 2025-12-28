# Group Management - Business Logic & Design

**Created**: 2025-12-28

Este documento centraliza la lógica de negocio completa del sistema de Group Management y las consideraciones de diseño que se están tomando en todas las specs relacionadas.

---

## 🔹 Concepto General

* **`Group`** es una entidad raíz.
* **No hay jerarquía** entre grupos (estructura plana).
* Existe un **grupo global** que contiene a todos los usuarios automáticamente.
* Los grupos pueden ser **visibles o no visibles**.
* Los grupos pueden definir **cómo se ingresa**:
  * Por **invitación** (solo admins invitan) - **requerido para grupos no visibles**
  * Por **solicitud** (usuarios piden, admins aprueban) - **solo para grupos visibles**
  * **Libre** (sin aprobación, ingreso automático) - **solo para grupos visibles**

---

## 🔹 Roles Dentro del Grupo

### Roles Disponibles
* Solo existen **dos roles**:
  * **Admin** (solo pueden ser coaches o system admin a nivel sistema)
  * **Member** (cualquier usuario)

### Permisos y Restricciones
* El **system admin** tiene permisos globales sobre todos los grupos.
* Un **coach** puede ser admin de uno o varios grupos.
* Un **usuario** puede pertenecer a múltiples grupos.
* Solo **coaches** y **system admins** pueden ser asignados como **Group Admin**.

---

## 🔹 Acceso y Visibilidad

| Estado del grupo | Usuario no miembro             | Políticas de ingreso permitidas |
| ---------------- | ------------------------------ | ------------------------------- |
| **Visible**      | Puede ver todo en modo lectura | `INVITE`, `REQUEST`, `OPEN`     |
| **No visible**   | No puede ver nada              | Solo `INVITE`                   |

### Reglas de Interacción
* Para **interactuar** (submissions, gestión, etc.) hay que ser **miembro**.
* Para **administrar**, hay que ser **admin del grupo**.
* Los no-miembros de grupos visibles pueden navegar contenido pero no participar.
* **Restricción importante**: Grupos no visibles solo pueden usar política `INVITE` (no tiene sentido `OPEN` o `REQUEST` si nadie puede ver el grupo).

---

## 🔹 Ingreso al Grupo

### Métodos de Ingreso
* **Invitaciones** y **solicitudes** solo pueden ser gestionadas por admins.
* **Ingreso libre** → el usuario entra automáticamente como miembro.
* El evento de ingreso se registra (ej. `joined_at`, método de ingreso).
* Un admin puede **cambiar el modo de ingreso** en cualquier momento.

### Flujos Específicos
1. **Por Invitación (`INVITE`)**:
   - Admin crea invitación con token UUID simple
   - Usuario acepta/rechaza usando el token
   - Invitaciones pueden tener fecha de expiración
   - **Soporte para invitación por username o email**

2. **Por Solicitud (`REQUEST`)**:
   - Usuario crea solicitud de ingreso
   - Admin aprueba/rechaza la solicitud
   - **Solo funciona en grupos visibles** (restricción de política)

3. **Ingreso Libre (`OPEN`)**:
   - Usuario se une directamente sin aprobación
   - **Solo funciona en grupos visibles** (restricción de política)
   - Ingreso inmediato como miembro

---

## 🔹 Contenido del Grupo

### Entidades Asociadas
* **Contests** y **materiales** **pertenecen al grupo**.
* **Problemas** son entidades **globales y reutilizables**.
* El **grupo global** puede tener materiales y contests públicos.

### Eliminación de Contenido
* Si se **elimina un grupo**:
  * Se eliminan **contests y materiales** (hard delete)
  * Los **problemas siguen existiendo** (son globales)
  * Las referencias históricas se conservan

---

## 🔹 Eliminación y Persistencia

### Eliminación de Grupos
* **Eliminación de grupo**: **hard delete**.
* **Excepción**: El grupo global **no puede eliminarse**.

### Gestión de Usuarios
* **Usuarios eliminados** del sistema → se **anonimizan** (según spec de user-management).
* Las **referencias históricas se conservan** (submissions, membership history).
* Los datos anónimos mantienen integridad referencial.

### Reglas de Membresía
* **Grupo global**: Los usuarios **no pueden salir** del grupo global.
* **Otros grupos**: Los usuarios pueden salir voluntariamente.
* **Último admin**: No se puede eliminar el último admin de un grupo.

---

## 🔹 Grupo Global (Default Group)

### Características Especiales
* **Creado automáticamente** durante bootstrap del sistema.
* **Todos los usuarios** son miembros automáticamente.
* **No se puede eliminar** ni modificar su membresía manualmente.
* **System admin** es admin del grupo global.
* Marcado con `is_default = true`.
* **Los usuarios pueden "ocultar" el grupo global** de su vista personal (soft hide) sin afectar la membresía real.

### Propósito
* Contenedor para contests y materiales públicos.
* Punto de referencia común para todos los usuarios.
* Facilita la gestión de contenido global del sistema.
* **Puede ocultarse de la UI** para usuarios que prefieren ver solo sus grupos específicos.

---

## 🔹 Consideraciones Técnicas

### Seguridad
* **Tokens de invitación**: UUID simples para facilitar implementación y debugging.
* **Validación estándar** de tokens UUID.
* **Tokens de un solo uso** (se invalidan al aceptar).
* **Invitaciones por username o email** para mayor flexibilidad.

### Concurrencia
* **Transacciones** para operaciones de membresía críticas.
* **Constraints de DB** para prevenir estados inconsistentes.
* **Validación atómica** para reglas como "último admin".

### Auditoría Selectiva
* **Solo cambios críticos** se registran para reducir overhead:
  * Creación/eliminación de grupos
  * Cambios de rol (Member ↔ Admin)
  * Adición/remoción de miembros por admin
  * Cambios de políticas de grupo (visibilidad, join policy)
* **No se auditan** operaciones menores como:
  * Aceptación de invitaciones (ya registrado en `joined_at`)
  * Listado de miembros o invitaciones
  * Operaciones de solo lectura

### Escalabilidad
* **Sin límites hard-coded** en número de miembros por grupo.
* **Índices apropiados** para consultas de membresía.
* **Paginación** en listados de miembros e invitaciones.

---

## 🔹 Specs Relacionadas

### Specs Implementadas
1. **[Create group](Create%20group/spec.md)** - Creación de grupos con configuración inicial
2. **[Join group](Join%20group/spec.md)** - Flujos de ingreso desde perspectiva del usuario
3. **[Invite to group](Invite%20to%20group/spec.md)** - Sistema de invitaciones
4. **[Manage group members](Manage%20group%20members/spec.md)** - Gestión administrativa de membresía

### Dependencias de Implementación
```
Create Group (base)
    ↓
Join Group (P1) ← Invite to Group (P2)
    ↓                    ↓
Manage Group Members (P2-P3)
```

### Specs Futuras (Consideradas)
* **Update Group** - Modificar metadatos, políticas de ingreso, visibilidad
* **Delete Group** - Eliminación segura con manejo de contenido asociado
* **Group Dashboard** - Vista "Mis Grupos" con filtros, búsqueda y gestión de visibilidad
* **Group Analytics** - Métricas de participación y actividad
* **Bulk Operations** - Gestión masiva de miembros e invitaciones

---

## 🔹 Confirmaciones Finales

### Nomenclatura
* **Nombre definitivo**: **Group** (no "Team", "Organization", etc.)
* **Consistencia** con el resto del sistema en naming y estructura.

### Alineación con Proyecto
* Las specs de grupos siguen **exactamente la misma estructura** que las demás specs.
* **No se inventan secciones nuevas** ni se rompe la consistencia.
* **Códigos de error** siguen las convenciones establecidas.

### Alcance del Spec de Creación
El spec de **Create Group** incluye:
* ✅ Creación del grupo con metadatos
* ✅ Asignación automática del creador como admin
* ✅ Configuración de políticas (visibilidad, ingreso)
* ✅ Relaciones iniciales (miembros/admins opcionales)
* ✅ Validaciones y restricciones de negocio

---

## � **cMejoras de Usabilidad y Mantenibilidad**

### **Dashboard "Mis Grupos"**
* **Vista centralizada** de todos los grupos del usuario
* **Filtros**: Por rol (Admin/Member), visibilidad, actividad reciente
* **Búsqueda**: Por nombre de grupo, descripción, tags
* **Gestión de visibilidad**: Opción de ocultar/mostrar grupo global
* **Acciones rápidas**: Salir de grupo, ver invitaciones pendientes

### **Invitaciones Mejoradas**
* **Por username**: Invitar usuarios conocidos sin necesidad de email
* **Por email**: Para usuarios externos que aún no tienen cuenta
* **Tokens UUID**: Simples, debuggeables, suficientemente seguros
* **Vista previa**: Mostrar información del grupo antes de aceptar

### **Políticas Simplificadas**
* **Combinaciones válidas reducidas**:
  * `VISIBLE + INVITE` ✅
  * `VISIBLE + REQUEST` ✅  
  * `VISIBLE + OPEN` ✅
  * `NOT_VISIBLE + INVITE` ✅
  * ~~`NOT_VISIBLE + REQUEST`~~ ❌ (sin sentido)
  * ~~`NOT_VISIBLE + OPEN`~~ ❌ (sin sentido)

### **Auditoría Inteligente**
* **Se registra**:
  * Creación/eliminación de grupos
  * Cambios de rol Admin ↔ Member
  * Adición/remoción manual de miembros
  * Cambios de políticas de grupo
* **No se registra**:
  * Aceptación de invitaciones (ya en `joined_at`)
  * Consultas de solo lectura
  * Operaciones automáticas del sistema

---

## 🔹 Decisiones de Diseño Clave

### ¿Por qué estructura plana?
* **Simplicidad**: Evita complejidad de permisos jerárquicos
* **Flexibilidad**: Los usuarios pueden estar en múltiples grupos independientes
* **Escalabilidad**: Más fácil de consultar y mantener

### ¿Por qué solo 2 roles?
* **Claridad**: Distinción simple entre administradores y miembros
* **Suficiencia**: Cubre todos los casos de uso identificados
* **Extensibilidad**: Se puede expandir en el futuro si es necesario

### ¿Por qué grupo global obligatorio?
* **Contenido público**: Lugar para contests/materiales accesibles a todos
* **Referencia común**: Todos los usuarios tienen al menos una membresía
* **Bootstrapping**: Facilita la inicialización del sistema

### ¿Por qué restricciones de política por visibilidad?
* **Lógica**: No tiene sentido que un grupo no visible permita ingreso libre (`OPEN`) o por solicitud (`REQUEST`)
* **Simplicidad**: Reduce combinaciones de 6 a 4 casos válidos
* **Usabilidad**: Evita configuraciones confusas para los administradores

### ¿Por qué tokens UUID simples?
* **Simplicidad**: Más fácil de implementar, debuggear y mantener
* **Suficiencia**: Para invitaciones internas, UUID provee suficiente entropía
* **Performance**: Generación y validación más rápida que tokens criptográficos complejos

### ¿Por qué auditoría selectiva?
* **Performance**: Reduce significativamente el volumen de logs
* **Mantenibilidad**: Menos datos que limpiar y gestionar
* **Enfoque**: Se concentra en cambios que realmente importan para compliance y debugging

### ¿Por qué opción de ocultar grupo global?
* **Usabilidad**: Usuarios avanzados pueden enfocarse en sus grupos específicos
* **Flexibilidad**: Mantiene la funcionalidad sin forzar la visibilidad
* **Adopción**: Facilita la transición para usuarios que prefieren interfaces limpias

---

*Este documento debe actualizarse cuando se tomen nuevas decisiones de diseño o se implementen specs adicionales.*