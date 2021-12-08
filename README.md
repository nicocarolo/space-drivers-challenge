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

## Introduction

The application implements a system to adminitrate users and its assigned travels developed following the
[Package-Oriented-Design](https://www.ardanlabs.com/blog/2017/02/package-oriented-design.html) guideline.

## Users

The application allows two kind of users: 'admin' and 'driver', to interact with the application users have to be
authenticated against [this resource](#authentication).

Only the `admin` user has the capability to create (write) and get (read) users. On the other hand, travels can be
created and modified by its owner and the admin user.

Travels can be modified by `drivers` only when they own it or if it still lacks an owner (driver would assign
itself to the travel).

### `POST` /v1/users

Create a user (only accessible by admins).

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
- role: the user role, must be `driver` or `admin`.

#### Response

`HTTP status code: 201`

```json
{
  "id": 3,
  "email": "driver2@hotmail.com",
  "role": "driver"
}
```

### `GET` /v1/users/:id

Get a user (only accessible by admins).

#### Response

`HTTP status code: 200`

```json
{
  "id": 3,
  "email": "driver2@hotmail.com",
  "role": "driver"
}
```

### `GET` /v1/users{?limit=n&offset=n}{?status=free}

Search driver users. The pagination search is only available for all drivers and not by status (as stated in exercise)

- status: search by driver status (`free` or `busy`, currently `busy` search is not working).
- limit: maximum quantity of users to obtain.
- offset: the number of records to skip before selecting drivers

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
- result: search matching drivers
- total: the total quantity of search drivers.

## Travel

Travels that have to be done by users (admin or drivers).

Attributes:

- status: `pending`, `in_process`, `ready`
- from: geolocation where the travel starts
    - latitude
    - longitude
- to: geolocation where the travel ends
    - latitude
    - longitude
- user_id: the user assigned to the travel

### `POST` /v1/travels

Create travel (only authorized for admin). The initial status for travel is `pending`.
Travels are created and assigned to a user from a `admin`, and the user assigned can be changed
(if the status still in pending).

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

### `GET` /v1/travels/:id

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

### `PUT` /v1/travels/:id

Update travel by id.

Validations:

- if the authenticated user is not the owner of the travel nor an admin then it cannot update the travel.
  - the travel is assigned to a driver from an admin.
- the travel location can´t be modified if its not in `pending` state.
- status can only be `pending`, `in_process`, `ready`.
- if the travel is not in `pending` status then the request should have a user id (the same user id already have).
- travels can have their user modified only when on pending state.
- status valid flow: `pending` → `in_process` → `ready`.

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

To access application resources users must be logged through `/v1/login`, if the email and password received are valid
application will return a token (JWT) with expiration date (20 minutes). If the user wants to access a resource, it
must send the received token in the request header Authorization.

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
using [migration.sql](database/migration.sql) (the initial status is with
an admin user, check credentials there on comment).

To monitor the app, we can observe metrics from the cloud services we use or our custom ones (Datadog):

- api health with traced endpoints by returned status code and elapsed time
  - `application.space.api.time`
  - `application.space.api.count`
- sql performance by entity (users and travels), operation, result and time
  - `application.space.repository.time`

App also logs errors (currently on stdout but can be indexed and used by services like Kibana).

It would be useful to add services like NewRelic to take more measurements like AppDex, custom transactions, services
tracing (storage), etc.

### Environment Variables

File `settings.env` holds db parameters and secrets used for the authentication token.

## Improvements

- Test repository with go sql mock.
- Usage of `select for update` on writes.
- Generalize a sql repository that can work with any model and move into /internal/platform.
- Add logout endpoint and refresh login token.
- Enhance JWT scheme dependency injection to improve unit tests.
- Add Metrics Provider (DataDog, New Relic)
- Enhance search by users role and drivers state (`busy` or `free`)