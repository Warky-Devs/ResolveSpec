package testmodels

import (
	"time"

	"github.com/Warky-Devs/ResolveSpec/pkg/modelregistry"
)

// Department represents a company department
type Department struct {
	ID          string    `json:"id" gorm:"primaryKey;type:string"`
	Name        string    `json:"name"`
	Code        string    `json:"code" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Employees []Employee `json:"employees,omitempty" gorm:"foreignKey:DepartmentID;references:ID"`
	Projects  []Project  `json:"projects,omitempty" gorm:"many2many:department_projects;"`
}

func (Department) TableName() string {
	return "departments"
}

// Employee represents a company employee
type Employee struct {
	ID           string    `json:"id" gorm:"primaryKey;type:string"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email" gorm:"uniqueIndex"`
	Title        string    `json:"title"`
	DepartmentID string    `json:"department_id" gorm:"type:string"`
	ManagerID    *string   `json:"manager_id" gorm:"type:string"`
	HireDate     time.Time `json:"hire_date"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Department *Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID;references:ID"`
	Manager    *Employee   `json:"manager,omitempty" gorm:"foreignKey:ManagerID;references:ID"`
	Reports    []Employee  `json:"reports,omitempty" gorm:"foreignKey:ManagerID;references:ID"`
	Projects   []Project   `json:"projects,omitempty" gorm:"many2many:employee_projects;"`
	Documents  []Document  `json:"documents,omitempty" gorm:"foreignKey:OwnerID;references:ID"`
}

func (Employee) TableName() string {
	return "employees"
}

// Project represents a company project
type Project struct {
	ID          string    `json:"id" gorm:"primaryKey;type:string"`
	Name        string    `json:"name"`
	Code        string    `json:"code" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Budget      float64   `json:"budget"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Departments []Department  `json:"departments,omitempty" gorm:"many2many:department_projects;"`
	Employees   []Employee    `json:"employees,omitempty" gorm:"many2many:employee_projects;"`
	Tasks       []ProjectTask `json:"tasks,omitempty" gorm:"foreignKey:ProjectID;references:ID"`
	Documents   []Document    `json:"documents,omitempty" gorm:"foreignKey:ProjectID;references:ID"`
}

func (Project) TableName() string {
	return "projects"
}

// ProjectTask represents a task within a project
type ProjectTask struct {
	ID          string    `json:"id" gorm:"primaryKey;type:string"`
	ProjectID   string    `json:"project_id" gorm:"type:string"`
	AssigneeID  string    `json:"assignee_id" gorm:"type:string"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	DueDate     time.Time `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Project  Project   `json:"project,omitempty" gorm:"foreignKey:ProjectID;references:ID"`
	Assignee Employee  `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID;references:ID"`
	Comments []Comment `json:"comments,omitempty" gorm:"foreignKey:TaskID;references:ID"`
}

func (ProjectTask) TableName() string {
	return "project_tasks"
}

// Document represents any document in the system
type Document struct {
	ID          string    `json:"id" gorm:"primaryKey;type:string"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Path        string    `json:"path"`
	OwnerID     string    `json:"owner_id" gorm:"type:string"`
	ProjectID   *string   `json:"project_id" gorm:"type:string"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Owner   Employee `json:"owner,omitempty" gorm:"foreignKey:OwnerID;references:ID"`
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;references:ID"`
}

func (Document) TableName() string {
	return "documents"
}

// Comment represents a comment on a task
type Comment struct {
	ID        string    `json:"id" gorm:"primaryKey;type:string"`
	TaskID    string    `json:"task_id" gorm:"type:string"`
	AuthorID  string    `json:"author_id" gorm:"type:string"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Task   ProjectTask `json:"task,omitempty" gorm:"foreignKey:TaskID;references:ID"`
	Author Employee    `json:"author,omitempty" gorm:"foreignKey:AuthorID;references:ID"`
}

func (Comment) TableName() string {
	return "comments"
}

// RegisterTestModels registers all test models with the provided registry
func RegisterTestModels(registry *modelregistry.DefaultModelRegistry) {
	registry.RegisterModel("departments", Department{})
	registry.RegisterModel("employees", Employee{})
	registry.RegisterModel("projects", Project{})
	registry.RegisterModel("project_tasks", ProjectTask{})
	registry.RegisterModel("documents", Document{})
	registry.RegisterModel("comments", Comment{})
}

// GetTestModels returns a list of all test model instances
func GetTestModels() []interface{} {
	return []interface{}{
		Department{},
		Employee{},
		Project{},
		ProjectTask{},
		Document{},
		Comment{},
	}
}
