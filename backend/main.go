package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/abstractmelon/robotregistry/backend/api"
	"github.com/abstractmelon/robotregistry/backend/database"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Load .env file from project root
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./robotregistry.db"
	}

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "../scraper/rce_data.json"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Initializing database at", dbPath)
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	shouldImport, err := db.ShouldImportData(dataPath)
	if err != nil {
		log.Printf("Warning: Failed to check if import is needed: %v\n", err)
		shouldImport = true
	}

	if shouldImport {
		log.Println("Importing data from", dataPath)
		if err := db.ImportData(dataPath); err != nil {
			log.Printf("Warning: Failed to import data: %v\n", err)
		}
	} else {
		log.Println("Database is up to date, skipping import")
	}

	router := mux.NewRouter()

	handler := api.NewHandler(db, dataPath)
	handler.RegisterRoutes(router)

	frontendPath := os.Getenv("FRONTEND_PATH")
	if frontendPath == "" {
		frontendPath = "../frontend/build"
	}

	if _, err := os.Stat(frontendPath); err == nil {
		log.Println("Serving frontend from", frontendPath)
		fs := http.FileServer(http.Dir(frontendPath))
		router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(frontendPath, r.URL.Path)
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(frontendPath, "index.html"))
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	handler_with_cors := corsHandler.Handler(router)

	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, handler_with_cors); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
