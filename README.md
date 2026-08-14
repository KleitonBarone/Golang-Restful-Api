# Golang Basic Restful Api

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.

### Updating OpenAPI spec

After modifying Swagger annotations in the Go source files, regenerate the spec:

```
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g router.go
```

## Getting Started

Run the following command:
```
go run main.go
```
