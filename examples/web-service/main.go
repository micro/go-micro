package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go-micro.dev/v5/web"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

var users = map[string]*User{
	"1": {ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
	"2": {ID: "2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
}

func main() {
	// Create a new web service
	service := web.NewService(
		web.Name("web.service"),
		web.Version("latest"),
		web.Address(":9090"),
	)

	// Initialize
	service.Init()

	// Register handlers
	service.HandleFunc("/", homeHandler)
	service.HandleFunc("/users", usersHandler)
	service.HandleFunc("/users/", userHandler)
	service.HandleFunc("/health", healthHandler)

	fmt.Println("Web service starting on :9090")
	fmt.Println("Try:")
	fmt.Println("  curl http://localhost:9090/")
	fmt.Println("  curl http://localhost:9090/users")
	fmt.Println("  curl http://localhost:9090/users/1")
	fmt.Println("  curl http://localhost:9090/health")

	// Run the service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "web.service",
		"version": "latest",
		"status":  "running",
	})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Return all users
	userList := make([]*User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}
	
	json.NewEncoder(w).Encode(userList)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Extract user ID from path
	id := r.URL.Path[len("/users/"):]
	
	user, exists := users[id]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}
	
	json.NewEncoder(w).Encode(user)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    "running",
	})
}
