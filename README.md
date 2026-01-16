# Golang Basic Restful Api

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.

### Updating OpenAPI spec

After modifying swagger annotations in `main.go`, regenerate the spec:

```
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

## Getting Started

Run the following command:
```
go run main.go
```