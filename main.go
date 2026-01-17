package main

import (
	"github.com/joho/godotenv"

	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"cms/config"
	"cms/handlers"
)

func main() {
	_ = godotenv.Load() // load .env file automatically
	config.LoadConfig("config.json")

	// Set up secure cookie store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET not set")
	}

	store := sessions.NewCookieStore([]byte(sessionSecret))
	handlers.SetStore(store)

	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/login", handlers.LoginForm).Methods("GET")
	r.HandleFunc("/login", handlers.Login).Methods("POST")
	r.HandleFunc("/logout", handlers.Logout).Methods("GET")

	// Static file server
	r.PathPrefix("/styles/").Handler(http.StripPrefix("/styles/", http.FileServer(http.Dir("./public/styles/"))))

	// Auth-protected routes
	protected := r.NewRoute().Subrouter()
	protected.Use(handlers.RequireLogin)

	// Dashboard
	protected.HandleFunc("/", handlers.Dashboard).Methods("GET")
	protected.HandleFunc("/dashboard", handlers.Dashboard).Methods("GET")

	// Generic content type routes
	protected.HandleFunc("/{type}/new", handlers.NewContentForm).Methods("GET")
	protected.HandleFunc("/{type}/edit/{slug}", handlers.EditContentForm).Methods("GET")
	protected.HandleFunc("/{type}/preview/{slug}", handlers.GetPreview).Methods("GET")
	protected.HandleFunc("/{type}", handlers.ListContent).Methods("GET")

	// Generic content API routes
	protected.HandleFunc("/api/{type}", handlers.CreateContent).Methods("POST")
	protected.HandleFunc("/api/{type}/{slug}", handlers.UpdateContent).Methods("PUT")
	protected.HandleFunc("/api/{type}/{slug}", handlers.DeleteContent).Methods("DELETE")

	// Shared upload endpoint
	protected.HandleFunc("/api/upload", handlers.UploadImage).Methods("POST")

	log.Println("CMS running on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
