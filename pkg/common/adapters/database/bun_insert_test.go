package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// TestInsertModel is a test model for insert operations
type TestInsertModel struct {
	bun.BaseModel `bun:"table:test_inserts"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Name          string `bun:"name,notnull"`
	Email         string `bun:"email"`
	Age           int    `bun:"age"`
}

func setupBunTestDB(t *testing.T) *bun.DB {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err, "Failed to open SQLite database")

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create test table
	_, err = db.NewCreateTable().
		Model((*TestInsertModel)(nil)).
		IfNotExists().
		Exec(context.Background())
	require.NoError(t, err, "Failed to create test table")

	return db
}

func TestBunInsertQuery_Model(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test inserting with Model()
	model := &TestInsertModel{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	result, err := adapter.NewInsert().
		Model(model).
		Returning("*").
		Exec(ctx)

	require.NoError(t, err, "Insert should succeed")
	assert.Equal(t, int64(1), result.RowsAffected(), "Should insert 1 row")

	// Verify the data was inserted
	var retrieved TestInsertModel
	err = db.NewSelect().
		Model(&retrieved).
		Where("id = ?", model.ID).
		Scan(ctx)

	require.NoError(t, err, "Should retrieve inserted row")
	assert.Equal(t, "John Doe", retrieved.Name)
	assert.Equal(t, "john@example.com", retrieved.Email)
	assert.Equal(t, 30, retrieved.Age)
}

func TestBunInsertQuery_Value(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test inserting with Value() method - this was the bug
	result, err := adapter.NewInsert().
		Table("test_inserts").
		Value("name", "Jane Smith").
		Value("email", "jane@example.com").
		Value("age", 25).
		Exec(ctx)

	require.NoError(t, err, "Insert with Value() should succeed")
	assert.Equal(t, int64(1), result.RowsAffected(), "Should insert 1 row")

	// Verify the data was inserted
	var retrieved TestInsertModel
	err = db.NewSelect().
		Model(&retrieved).
		Where("name = ?", "Jane Smith").
		Scan(ctx)

	require.NoError(t, err, "Should retrieve inserted row")
	assert.Equal(t, "Jane Smith", retrieved.Name)
	assert.Equal(t, "jane@example.com", retrieved.Email)
	assert.Equal(t, 25, retrieved.Age)
}

func TestBunInsertQuery_MultipleValues(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test inserting multiple values
	result, err := adapter.NewInsert().
		Table("test_inserts").
		Value("name", "Alice").
		Value("email", "alice@example.com").
		Value("age", 28).
		Exec(ctx)

	require.NoError(t, err, "First insert should succeed")
	assert.Equal(t, int64(1), result.RowsAffected())

	result, err = adapter.NewInsert().
		Table("test_inserts").
		Value("name", "Bob").
		Value("email", "bob@example.com").
		Value("age", 35).
		Exec(ctx)

	require.NoError(t, err, "Second insert should succeed")
	assert.Equal(t, int64(1), result.RowsAffected())

	// Verify both rows exist
	var count int
	count, err = db.NewSelect().
		Model((*TestInsertModel)(nil)).
		Count(ctx)

	require.NoError(t, err, "Count should succeed")
	assert.Equal(t, 2, count, "Should have 2 rows")
}

func TestBunInsertQuery_ValueWithNil(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test inserting with nil value for nullable field
	result, err := adapter.NewInsert().
		Table("test_inserts").
		Value("name", "Test User").
		Value("email", nil). // NULL email
		Value("age", 20).
		Exec(ctx)

	require.NoError(t, err, "Insert with nil value should succeed")
	assert.Equal(t, int64(1), result.RowsAffected())

	// Verify the data was inserted with NULL email
	var retrieved TestInsertModel
	err = db.NewSelect().
		Model(&retrieved).
		Where("name = ?", "Test User").
		Scan(ctx)

	require.NoError(t, err, "Should retrieve inserted row")
	assert.Equal(t, "Test User", retrieved.Name)
	assert.Equal(t, "", retrieved.Email) // NULL becomes empty string
	assert.Equal(t, 20, retrieved.Age)
}

func TestBunInsertQuery_Returning(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test insert with RETURNING clause
	// Note: SQLite has limited RETURNING support, but this tests the API
	result, err := adapter.NewInsert().
		Table("test_inserts").
		Value("name", "Return Test").
		Value("email", "return@example.com").
		Value("age", 40).
		Returning("*").
		Exec(ctx)

	require.NoError(t, err, "Insert with RETURNING should succeed")
	assert.Equal(t, int64(1), result.RowsAffected())
}

func TestBunInsertQuery_EmptyValues(t *testing.T) {
	db := setupBunTestDB(t)
	defer db.Close()

	adapter := NewBunAdapter(db)
	ctx := context.Background()

	// Test insert without calling Value() - should use Model() or fail gracefully
	result, err := adapter.NewInsert().
		Table("test_inserts").
		Exec(ctx)

	// This should fail because no values are provided
	assert.Error(t, err, "Insert without values should fail")
	if result != nil {
		assert.Equal(t, int64(0), result.RowsAffected())
	}
}
