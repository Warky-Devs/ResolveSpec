package test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	TestSetup(m)
}

func TestDepartmentEmployees(t *testing.T) {
	// Create test department
	deptPayload := map[string]interface{}{
		"operation": "create",
		"data": map[string]interface{}{
			"id":          "dept1",
			"name":        "Engineering",
			"code":        "ENG",
			"description": "Engineering Department",
		},
	}

	resp := makeRequest(t, "/test/departments", deptPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Create employees in department
	empPayload := map[string]interface{}{
		"operation": "create",
		"data": []map[string]interface{}{
			{
				"id":            "emp1",
				"first_name":    "John",
				"last_name":     "Doe",
				"email":         "john@example.com",
				"department_id": "dept1",
				"title":         "Senior Engineer",
			},
			{
				"id":            "emp2",
				"first_name":    "Jane",
				"last_name":     "Smith",
				"email":         "jane@example.com",
				"department_id": "dept1",
				"title":         "Engineer",
			},
		},
	}

	resp = makeRequest(t, "/test/employees", empPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read department with employees
	readPayload := map[string]interface{}{
		"operation": "read",
		"options": map[string]interface{}{
			"preload": []map[string]interface{}{
				{
					"relation": "employees",
					"columns":  []string{"id", "first_name", "last_name", "title"},
				},
			},
		},
	}

	resp = makeRequest(t, "/test/departments/dept1", readPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data := result["data"].(map[string]interface{})
	employees := data["employees"].([]interface{})
	assert.Equal(t, 2, len(employees))
}

func TestEmployeeHierarchy(t *testing.T) {
	// Create manager
	mgrPayload := map[string]interface{}{
		"operation": "create",
		"data": map[string]interface{}{
			"id":            "mgr1",
			"first_name":    "Alice",
			"last_name":     "Manager",
			"email":         "alice@example.com",
			"title":         "Engineering Manager",
			"department_id": "dept1",
		},
	}

	resp := makeRequest(t, "/test/employees", mgrPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Update employees to set manager
	updatePayload := map[string]interface{}{
		"operation": "update",
		"data": map[string]interface{}{
			"manager_id": "mgr1",
		},
	}

	resp = makeRequest(t, "/test/employees/emp1", updatePayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp = makeRequest(t, "/test/employees/emp2", updatePayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read manager with reports
	readPayload := map[string]interface{}{
		"operation": "read",
		"options": map[string]interface{}{
			"preload": []map[string]interface{}{
				{
					"relation": "reports",
					"columns":  []string{"id", "first_name", "last_name", "title"},
				},
			},
		},
	}

	resp = makeRequest(t, "/test/employees/mgr1", readPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data := result["data"].(map[string]interface{})
	reports := data["reports"].([]interface{})
	assert.Equal(t, 2, len(reports))
}

func TestProjectStructure(t *testing.T) {
	// Create project
	projectPayload := map[string]interface{}{
		"operation": "create",
		"data": map[string]interface{}{
			"id":          "proj1",
			"name":        "New Website",
			"code":        "WEB",
			"description": "Company website redesign",
			"status":      "active",
			"start_date":  time.Now().Format(time.RFC3339),
			"end_date":    time.Now().AddDate(0, 3, 0).Format(time.RFC3339),
			"budget":      100000,
		},
	}

	resp := makeRequest(t, "/test/projects", projectPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Create project tasks
	taskPayload := map[string]interface{}{
		"operation": "create",
		"data": []map[string]interface{}{
			{
				"id":          "task1",
				"project_id":  "proj1",
				"assignee_id": "emp1",
				"title":       "Design Homepage",
				"description": "Create homepage design",
				"status":      "in_progress",
				"priority":    1,
				"due_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
			},
			{
				"id":          "task2",
				"project_id":  "proj1",
				"assignee_id": "emp2",
				"title":       "Implement Backend",
				"description": "Implement backend APIs",
				"status":      "planned",
				"priority":    2,
				"due_date":    time.Now().AddDate(0, 2, 0).Format(time.RFC3339),
			},
		},
	}

	resp = makeRequest(t, "/test/project_tasks", taskPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Create task comments
	commentPayload := map[string]interface{}{
		"operation": "create",
		"data": map[string]interface{}{
			"id":        "comment1",
			"task_id":   "task1",
			"author_id": "mgr1",
			"content":   "Looking good! Please add more animations.",
		},
	}

	resp = makeRequest(t, "/test/comments", commentPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read project with all relations
	readPayload := map[string]interface{}{
		"operation": "read",
		"options": map[string]interface{}{
			"preload": []map[string]interface{}{
				{
					"relation": "tasks",
					"columns":  []string{"id", "title", "status", "assignee_id"},
					"preload": []map[string]interface{}{
						{
							"relation": "comments",
							"columns":  []string{"id", "content", "author_id"},
							"preload": []map[string]interface{}{
								{
									"relation": "author",
									"columns":  []string{"id", "first_name", "last_name"},
								},
							},
						},
						{
							"relation": "assignee",
							"columns":  []string{"id", "first_name", "last_name", "title"},
						},
					},
				},
			},
		},
	}

	resp = makeRequest(t, "/test/projects/proj1", readPayload)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.True(t, result["success"].(bool))
}
