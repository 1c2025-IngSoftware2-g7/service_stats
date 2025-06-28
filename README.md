# ClassConnect - API Stats

## Contenidos
1. Introducción
2. Arquitectura de componentes
3. Pre-requisitos y detalles de arquitectura
4. CI-CD
5. Tests
6. Comandos para construir la imagen de Docker
7. Comandos para correr la base de datos
8. Comandos para correr la imagen del servicio
9. Despliegue en la Nube 
10. Analisis postmortem

## 1. Introducción

Microservicio para la gestión de Estadísticas en ClassConnect.

El servicio permite:
    - Insertar estadísticas para un estudiante/curso
    - Obtención de estádisticas para un estudiante/curso.
    - Utilización de Colas para la ejecución de tareas. (Punto opcional del Trabajo Práctico)
    
## 2. Arquitectura de componentes

El servicio consta de:
    - Base de datos (PostgreSQL) : Donde se guardan las estadísticas tanto de usuarios/estudiantes
    - Redis: Se utiliza para la ejecución de cola de tareas
    - api main: Donde se hace un handler de cada endpoint para la obtencion/insersión de estádisticas
    - Worker: Aplicación exclusiva que se encarga de procesar las tareas previamente

## 3. Pre-requisitos y detalles de arquitectura
- Necesario para levantar el entorno de desarrollo de forma local:
    - [Docker](https://docs.docker.com/get-started/introduction/) (version 27.3.1) 
    - [Docker-compose](https://docs.docker.com/compose/install/) (version 2.30.3)
    - [Asynqmon](https://github.com/hibiken/asynqmon) (Version 0.7.2)

- Puertos utilizados: 
    - 5432: Utilizado por la base de datos PostgreSQL.
    - 8080: Utilizado por la API.
    - 2222: Utilizado por Asynqmon (Visibilidad de la queue)

Adicionalmente, se menciona a continuación lo utilizado dentro de los contenedores:

- Lenguaje:
    - Golang 1.14 (Utilizado en la imagen del Dockerfile).

- Base de datos:
    - PostgreSQL 14 (imagen oficial).

En este microservicio intentamos ir por la idea de arquitectura de capas, donde la responsabilidad está distribuida, y sea la api o worker quien desea consumirlas, pueden acceder sin impedimentos.


## 4. CI-CD

[![codecov](https://codecov.io/gh/1c2025-IngSoftware2-g7/service_stats/branch/feature%2Ftests/graph/badge.svg?token=51BR8Q143V)](https://codecov.io/gh/1c2025-IngSoftware2-g7/service_stats)

### Test coverage


## 5. Tests

Para correr los tests, se puede realizar con el comando

``` go test -coverprofile=coverage.out -covermode=atomic ./internal/... ``` 

Aclaración: Con esto se skipea los directorios donde tiene los archivos ejecutables

## 6. Comandos para construir la imagen de Docker
Al utilizar docker-compose, se puede construir todas las imágenes definidas en docker-compose.yml con el siguiente comando:
```bash
docker compose build
```

## 7. Comandos para correr la base de datos
Como ya se mencionó, se utilizó docker compose para correr el servicio de forma local. Por lo que para levantar todas las imágenes del proyecto, se debe correr:
```bash
docker compose up
```

En ```docker-compose.yml```:
- db local: Base de datos PostgreSQL. Se define la imagen oficial, los parámetros para la conexión, el puerto en el que escuchará (5432) y se carga el script que se debe correr para inicializar la base de datos, un tipo de usuario, sus permisos y se crea la tabla de cursos. Además, se define la red a la que va a pertenecer.  

- el servicio utiliza Redis para la queue de taraeas.

## 8. Comandos para correr la imagen del servicio
De igual forma que en el inciso anterior:
```bash
docker compose up
```

En ```docker-compose.yml```:
- app: API RESTful en Flask. Se utiliza como imagen la definida en Dockerfile. Se indica el puerto 8080 para comunicarse con este servicio y se incluye en la misma red que la base de datos, de esta forma se pueden comunicar. Además, se define que este servicio se va a correr cuando se termine de levantar la base de datos. Por último, se indica el comando que se va a correr.


## 9. Despliegue en la Nube 

TBD

## 10. Analisis postmortem

A explicar mas llegado el momento.