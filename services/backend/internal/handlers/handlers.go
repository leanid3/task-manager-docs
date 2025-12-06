package handlers

import (
	"app/internal/models"
	"app/pkg/response"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		response.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"message": "Welcome to MyApp API",
		"version": "1.0.0",
		"endpoints": []string{
			"GET /health",
			"GET /api/users",
			"GET /api/users/{id}",
			"POST /api/users",
		},
	}
	response.JSON(w, data, http.StatusOK)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status": "healthy",
	}
	response.JSON(w, data, http.StatusOK)
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users := models.GetAllUsers()
		response.JSON(w, users, http.StatusOK)

	case http.MethodPost:
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Email == "" {
			response.Error(w, "Name and email are required", http.StatusBadRequest)
			return
		}

		user := models.CreateUser(req.Name, req.Email)
		response.JSON(w, user, http.StatusCreated)

	default:
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func UserByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из URL: /api/users/1
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 {
		response.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		response.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := models.GetUserByID(id)
	if err != nil {
		response.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response.JSON(w, user, http.StatusOK)
}
