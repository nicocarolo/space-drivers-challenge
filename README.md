# Space Drivers

## Enunciado

La API es usada por una empresa de un servicio de mercadería que es repartida por conductores. Esta API permite ingresar
como administrador, gestionar el ABM de los conductores y ver un listado de ellos.

- Crear un proyecto de go usando módulos y tu librería HTTP de preferencia (en Space Guru usamos gin)
- Crear una aplicación para gestionar los conductores que debe permitir:
    - Autenticar y autorizar usuarios admin
    - Autenticar conductores
    - Crear un nuevo conductor (junto con las credenciales de autenticación del mismo)
    - Obtener todos los conductores con paginación
    - Obtener todos los conductores que no estén realizando un viaje (tenés la libertad de elegir cómo identificar si
      ese conductor ya se encuentra realizando un viaje)
- Cada ruta debe autenticarse vía un middleware que use el esquema de autenticación de preferencia (jwt, basic auth,
  custom token, etc)
- Usar una base de datos relacional y agregar en un archivo .sql los scripts de creación de las tablas que uses (ej.
  mysql)
- El servicio debe considerar los siguientes principios de diseño para su implementación:
    - Manejo de código de estado 5xx y panics
    - Al menos un patrón de diseño de estructura como (DDD, MVC, etc)
    - Principios REST (códigos de estados y verbos correcto así como convención de rutas)
    - Separar la capa de presentación y datos (requerimiento mínimo)-
- Al menos un endpoint debe tener pruebas unitarias, el resto son solo necesarias si querés demostrar tus capacidades

Introduction

This application is about users who want to administrate drivers and travels that has to be done. It is developed trying
to follow [Package-Oriented-Design](https://www.ardanlabs.com/blog/2017/02/package-oriented-design.html) guideline

## Users

The application allow two kind of users: 'admin' and 'driver', to interact with the application the user has to be
logged in on [this resource](#authentication).

A `admin` user has the capabilities to create more users (write) and get anyone (read) and is the only one able to
create travels and modify it even if he is not the owner (travel must be in pending status). A `driver` user can modify
travel without user assigned or if he is the one assigned.

### `POST` /v1/user/

Create a user (only authorized for admin).

#### Request

```json
{
  "email": "driver2@hotmail.com",
  "password": "hola1234",
  "role": "driver"
}
```

- email: the user email.
- password: the user password.
- role: the user role, must be `driver` or `role`.

#### Response

`HTTP status code: 201`

```json
{
  "id": 3,
  "email": "driver2@hotmail.com",
  "role": "driver"
}
```

### `GET` /v1/user/{id}

Get a user (only authorized for admin).

#### Request

```json
{}
```

#### Response

`HTTP status code: 200`

```json
{
  "id": 3,
  "email": "driver2@hotmail.com",
  "role": "driver"
}
```

### `GET` /v1/user/drivers{?status=free}{?limit=20&offset=2}

Search driver users (only authorized for admin). The pagination search is only available for all drivers (not by status)
.

- status: search for driver status, could be `free` or `busy` (currently `busy` search is not working).
- limit: maximum quantity of users to obtain.
- offset: step to move index on start driver to return.

#### Response

`HTTP status code: 200`

```json
{
  "pending": 1,
  "result": [
    {
      "id": 1,
      "email": "nico.carolo@hotmail.com",
      "role": "admin"
    }
  ],
  "total": 2
}
```

- pending: users pending to get.
- result: the drivers.
- total: the quantity of driver on storage.

## Travel

Travel that has to be done for users (admin or drivers).

Attributes:

- status: `pending`, `in_process`, `ready`
- from: geolocation where the travel starts
    - latitude
    - longitude
- to: geolocation where the travel ends
    - latitude
    - longitude
- user_id: the user assigned to the travel

### `POST` /v1/travel/

Create travel (only authorized for admin). The initial status for travel is `pending`.

#### Request

```json
{
  "from": {
    "latitude": 1.12312,
    "longitude": 2
  },
  "to": {
    "latitude": -1,
    "longitude": -2.02
  },
  "user_id": 3
}
```

#### Response

`HTTP status code: 201`

```json
{
  "id": 5,
  "status": "pending",
  "from": {
    "latitude": 1.12312,
    "longitude": 2
  },
  "to": {
    "latitude": -1,
    "longitude": -2.02
  },
  "user_id": 3
}
```

### `GET` /v1/travel/{id}

Get travel by id

#### Response

`HTTP status code: 200`

```json
{
  "id": 5,
  "status": "pending",
  "from": {
    "latitude": 1.12312,
    "longitude": 2
  },
  "to": {
    "latitude": -1,
    "longitude": -2.02
  },
  "user_id": 3
}
```

### `PUT` /v1/travel/{id}

Update travel by id.

Validations:

- if the user who is logged is not the owner of the travel, and it is not an admin then it cannot update travel.
- it cannot change the location if the travel is not in `pending` status.
- status can only be `pending`, `in_process`, `ready`.
- if the travel is not in `pending` status then the request should have a user id (the same user id already have).
- if there is a change on the user id, when the travel already have a user, then the status received it should be
  pending.
- status changes must be: `pending` → `in_process` → `ready`.

#### Request

```json
{
  "status": "pending",
  "from": {
    "latitude": 1.12312,
    "longitude": 2
  },
  "to": {
    "latitude": -1,
    "longitude": -2.02
  },
  "user_id": 3
}
```

#### Response

`HTTP status code: 200`

```json
{
  "id": 5,
  "status": "pending",
  "from": {
    "latitude": 1.12312,
    "longitude": 2
  },
  "to": {
    "latitude": -1,
    "longitude": -2.02
  },
  "user_id": 3
}
```

## Authentication

To access application resources users must log in through `/v1/login`, if the email and password received are valid
application will return a token (JWT) with expiration date (20 minutes). Then, if the user want to access to any
resource, it must send the received token as header on the request.

```
Authorization: Bearer {{token}}
```

### `POST` `/v1/login`

#### Request

```json
{
  "email": "an_email@hotmail.com",
  "password": "a_password"
}
```

#### Response

`HTTP status code: 200`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2Mzg4NDY1MzEsImlhdCI6MTYzODg0NTMzMSwicm9sZSI6ImFkbWluIiwidXNlcl9pZCI6MX0.e9X1dOHS_-DH7ShXQ7PbzzgbNH86I4yeXmM3EDRowvE"
}
```

## Errors

- User
    - 400: `invalid_password`: `cannot assign received password to user`
    - 500: `storage_failure`: `an error ocurred trying to save user`
    - 500: `storage_failure`: `an error ocurred trying to get user`
    - 404: `not_found_user`: `not founded the user to get`
    - 400: `invalid_role`: `the received role should be admin or driver`
- Authentication
    - 400: `invalid_password`: `the password received to login is invalid`
    - 404: `not_found_user`: `not founded the user to get`
    - 500: `storage_failure`: `an error ocurred trying to get user`
    - 401: `authorization_token_missing`: `it was not received the authorization header with token`
    - 401: `expired_token`
    - 401: `invalid_token`
    - 401: `invalid_token_data`
- Travel
    - 500: `storage_failure`: `an error ocurred trying to save travel`
    - 500: `storage_failure`: `an error ocurred trying to update travel`
    - 500: `storage_failure`: `an error ocurred trying to get travel`
    - 404: `not_found_travel`: `not founded the travel to get`
    - 400: `invalid_location_edit_status`: `travel status does not allow location change`
    - 400: `invalid_status`: `invalid received status`
    - 400: `invalid_user`: `invalid user while performing update`
    - 401: `invalid_user_access`: `cannot identify user logged in`
    - 401: `invalid_user_access`: `the user logged in cannot perform this action, he is not the owner of the travel and it is not an admin`

## Deployment

To run the application, you must execute on the project root.

```bash
docker-compose up --build
```

The docker configuration will start two containers: sql and application. The mysql database will be configured
using [migration.sql](database/migration.sql).

To monitoring the app, we can observe metrics from cloud service that we use or the added from us (mock tracing with
service like Datadog) where we can track:

- api health with traced endpoints with status code returned and time elapsed
    - `application.space.api.time`
    - `application.space.api.count`
- sql performance by entity (users and travels), operation, result and time
    - `application.space.repository.time`

The app also log the errors (currently on stdout but can be indexed and use services like Kibana for the availability).
It could be nice to add services like NewRelic to take more measurements from an AppDex, custom transactions, services
tracing (storage).

On `settings.env` it can set db parameters to connect the app and `jwt_secret` used for the authentication token.

## Improvements

- Use uid for entity id and no mysql autoincrement.
- Test repository with go sql mock.
- Work with `select for update` on write sql db.
- Generalize a sql repository that can work with any model and move into /internal/platform.
- Add logout endpoint and refresh login token.
- Add interface for jwt to work with dependency injection to make it able to test.
- Add metrics client.
- Improve search and add `busy` drivers search.