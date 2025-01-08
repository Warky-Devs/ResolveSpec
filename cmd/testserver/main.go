package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
	"github.com/Warky-Devs/ResolveSpec/pkg/models"
	"github.com/Warky-Devs/ResolveSpec/pkg/testmodels"

	"github.com/Warky-Devs/ResolveSpec/pkg/resolvespec"
	"github.com/gorilla/mux"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

func main() {
	// Initialize logger
	fmt.Println("ResolveSpec test server starting")
	logger.Init(true)

	// Init Models
	testmodels.RegisterTestModels()

	// Initialize database
	db, err := initDB()
	if err != nil {
		logger.Error("Failed to initialize database: %+v", err)
		os.Exit(1)
	}

	// Create router
	r := mux.NewRouter()

	// Initialize API handler
	handler := resolvespec.NewAPIHandler(db)

	// Setup routes
	r.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.Handle(w, r, vars)
	}).Methods("POST")

	r.HandleFunc("/{schema}/{entity}/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.Handle(w, r, vars)
	}).Methods("POST")

	r.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.HandleGet(w, r, vars)
	}).Methods("GET")

	// Start server
	logger.Info("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Error("Server failed to start: %v", err)
		os.Exit(1)
	}
}

func initDB() (*gorm.DB, error) {

	newLogger := gormlog.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gormlog.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  gormlog.Info, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,         // Don't include params in the SQL log
			Colorful:                  true,         // Disable color
		},
	)

	// Create SQLite database
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{Logger: newLogger, FullSaveAssociations: false})
	if err != nil {
		return nil, err
	}

	modelList := models.GetModels()

	// Auto migrate schemas
	err = db.AutoMigrate(modelList...)
	if err != nil {
		return nil, err
	}

	return db, nil
}
