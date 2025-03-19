package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type User struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func connectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ DATABASE_URL environment variable is not set")
	}

	fmt.Println("🔍 Connecting to DB...")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}

	fmt.Println("✅ Connected to PostgreSQL!")
	db.AutoMigrate(&User{})
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []User
	if result := db.Find(&users); result.Error != nil {
		http.Error(w, `{"error": "Failed to retrieve users"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Validation
	if user.Name == "" {
		http.Error(w, `{"error": "Name is required"}`, http.StatusBadRequest)
		return
	}

	if user.Email == "" || !isValidEmail(user.Email) {
		http.Error(w, `{"error": "Invalid email format"}`, http.StatusBadRequest)
		return
	}

	if result := db.Create(&user); result.Error != nil {
		http.Error(w, `{"error": "Failed to create user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid user ID"}`, http.StatusBadRequest)
		return
	}

	var user User
	if result := db.First(&user, id); result.Error != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	var updateData User
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Validation
	if updateData.Name != "" && len(updateData.Name) < 3 {
		http.Error(w, `{"error": "Name must be at least 3 characters"}`, http.StatusBadRequest)
		return
	}

	if updateData.Email != "" && !isValidEmail(updateData.Email) {
		http.Error(w, `{"error": "Invalid email format"}`, http.StatusBadRequest)
		return
	}

	// Only update fields that are provided
	if updateData.Name != "" {
		user.Name = updateData.Name
	}
	if updateData.Email != "" {
		user.Email = updateData.Email
	}

	db.Save(&user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid user ID"}`, http.StatusBadRequest)
		return
	}

	if result := db.Delete(&User{}, id); result.Error != nil {
		http.Error(w, `{"error": "Failed to delete user"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "✅ Welcome to my Go API! Available endpoints: GET/POST/PUT/DELETE /api/users")
}

func main() {
	connectDB()

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/api/users", getUsers).Methods("GET")
	r.HandleFunc("/api/users", createUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", updateUser).Methods("PUT")
	r.HandleFunc("/api/users/{id}", deleteUser).Methods("DELETE")

	port := "8080"
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		fmt.Println("🚀 Server is running on http://localhost:" + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Handle shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("\n🛑 Shutting down server gracefully...")

	// Close database connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ Error getting DB connection: %v", err)
	}
	sqlDB.Close()
	fmt.Println("✅ Database connection closed")
}
