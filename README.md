# Golang Basic Restful Api

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.

### Updating OpenAPI spec

After modifying Swagger annotations in the Go source files, regenerate the spec:

```
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g router.go
```

## Getting Started

Run the service:

```
go run .
```

It listens on `localhost:8080` by default. Set `LISTEN_ADDRESS` to bind to a
different address, such as `:9090`.

Use `GET /health` for local and automated readiness checks. A healthy process
returns `200 OK` with `{"status":"ok"}`.

On an interrupt or termination signal, the server stops accepting new
connections and gives in-flight requests up to five seconds to finish.
The server allows at most ten seconds to read each complete request and ten
seconds to write its response. Request headers are limited to 16 KiB.
