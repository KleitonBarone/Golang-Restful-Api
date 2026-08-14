package main

func main() {
	router := setupRouter()
	router.Run("localhost:8080")
}
