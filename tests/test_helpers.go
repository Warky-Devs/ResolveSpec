package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bitechdev/ResolveSpec/pkg/logger"
	"github.com/bitechdev/ResolveSpec/pkg/modelregistry"
	"github.com/bitechdev/ResolveSpec/pkg/resolvespec"
	"github.com/bitechdev/ResolveSpec/pkg/testmodels"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"gorm.io/gorm"
)

var (
	testDB        *gorm.DB
	testServer    *httptest.Server
	testServerURL string
)

// makeRequest is a helper function to make HTTP requests in tests
func makeRequest(t *testing.T, path string, payload interface{}) *http.Response {
	jsonData, err := json.Marshal(payload)
	assert.NoError(t, err, "Failed to marshal request payload")

	logger.Debug("Making request to %s with payload: %s", path, string(jsonData))

	req, err := http.NewRequest("POST", testServerURL+path, bytes.NewBuffer(jsonData))
	assert.NoError(t, err, "Failed to create request")

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Failed to execute request")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("Request failed with status %d: %s", resp.StatusCode, string(body))
	} else {
		logger.Debug("Request successful with status %d", resp.StatusCode)
	}

	return resp
}

// verifyResponse is a helper function to verify response status and decode body
func verifyResponse(t *testing.T, resp *http.Response) map[string]interface{} {
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected response status")

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err, "Failed to decode response")
	assert.True(t, result["success"].(bool), "Response indicates failure")

	return result
}

// TestSetup initializes the test environment
func TestSetup(m *testing.M) int {
	logger.Init(true)

	logger.Info("Setting up test environment")

	// Create test database
	db, err := setupTestDB()
	if err != nil {
		logger.Error("Failed to setup test database: %v", err)
		return 1
	}
	testDB = db

	// Setup test server
	router := setupTestRouter(testDB)
	testServer = httptest.NewServer(router)

	logger.Info("ResolveSpec test server starting on  %s", testServer.URL)
	testServerURL = testServer.URL

	defer testServer.Close()

	// Run tests
	code := m.Run()

	// Cleanup
	logger.Info("Cleaning up test environment")
	cleanup()
	return code
}

// setupTestDB creates and initializes the test database
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Auto migrate all test models
	err = autoMigrateModels(db)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate models: %v", err)
	}

	return db, nil
}

// setupTestRouter creates and configures the test router
func setupTestRouter(db *gorm.DB) http.Handler {
	r := mux.NewRouter()

	// Create a new registry instance
	registry := modelregistry.NewModelRegistry()

	// Register test models with the registry
	testmodels.RegisterTestModels(registry)

	// Create handler with GORM adapter and the registry
	handler := resolvespec.NewHandlerWithGORM(db)

	// Register test models with the handler for the "test" schema
	models := testmodels.GetTestModels()
	modelNames := []string{"departments", "employees", "projects", "project_tasks", "documents", "comments"}
	for i, model := range models {
		handler.RegisterModel("test", modelNames[i], model)
	}

	resolvespec.SetupMuxRoutes(r, handler)

	return r
}

// cleanup performs test cleanup
func cleanup() {
	if testDB != nil {
		db, err := testDB.DB()
		if err == nil {
			db.Close()
		}
	}
	os.Remove("test.db")
}

// autoMigrateModels performs automigration for all test models
func autoMigrateModels(db *gorm.DB) error {
	modelList := testmodels.GetTestModels()
	return db.AutoMigrate(modelList...)
}
