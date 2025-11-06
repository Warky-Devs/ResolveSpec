package database

import (
	"context"

	"github.com/Warky-Devs/ResolveSpec/pkg/common"
	"gorm.io/gorm"
)

// GormAdapter adapts GORM to work with our Database interface
type GormAdapter struct {
	db *gorm.DB
}

// NewGormAdapter creates a new GORM adapter
func NewGormAdapter(db *gorm.DB) *GormAdapter {
	return &GormAdapter{db: db}
}

func (g *GormAdapter) NewSelect() common.SelectQuery {
	return &GormSelectQuery{db: g.db}
}

func (g *GormAdapter) NewInsert() common.InsertQuery {
	return &GormInsertQuery{db: g.db}
}

func (g *GormAdapter) NewUpdate() common.UpdateQuery {
	return &GormUpdateQuery{db: g.db}
}

func (g *GormAdapter) NewDelete() common.DeleteQuery {
	return &GormDeleteQuery{db: g.db}
}

func (g *GormAdapter) Exec(ctx context.Context, query string, args ...interface{}) (common.Result, error) {
	result := g.db.WithContext(ctx).Exec(query, args...)
	return &GormResult{result: result}, result.Error
}

func (g *GormAdapter) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return g.db.WithContext(ctx).Raw(query, args...).Find(dest).Error
}

func (g *GormAdapter) BeginTx(ctx context.Context) (common.Database, error) {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &GormAdapter{db: tx}, nil
}

func (g *GormAdapter) CommitTx(ctx context.Context) error {
	return g.db.WithContext(ctx).Commit().Error
}

func (g *GormAdapter) RollbackTx(ctx context.Context) error {
	return g.db.WithContext(ctx).Rollback().Error
}

func (g *GormAdapter) RunInTransaction(ctx context.Context, fn func(common.Database) error) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		adapter := &GormAdapter{db: tx}
		return fn(adapter)
	})
}

// GormSelectQuery implements SelectQuery for GORM
type GormSelectQuery struct {
	db *gorm.DB
}

func (g *GormSelectQuery) Model(model interface{}) common.SelectQuery {
	g.db = g.db.Model(model)
	return g
}

func (g *GormSelectQuery) Table(table string) common.SelectQuery {
	g.db = g.db.Table(table)
	return g
}

func (g *GormSelectQuery) Column(columns ...string) common.SelectQuery {
	g.db = g.db.Select(columns)
	return g
}

func (g *GormSelectQuery) Where(query string, args ...interface{}) common.SelectQuery {
	g.db = g.db.Where(query, args...)
	return g
}

func (g *GormSelectQuery) WhereOr(query string, args ...interface{}) common.SelectQuery {
	g.db = g.db.Or(query, args...)
	return g
}

func (g *GormSelectQuery) Join(query string, args ...interface{}) common.SelectQuery {
	g.db = g.db.Joins(query, args...)
	return g
}

func (g *GormSelectQuery) LeftJoin(query string, args ...interface{}) common.SelectQuery {
	g.db = g.db.Joins("LEFT JOIN "+query, args...)
	return g
}

func (g *GormSelectQuery) Preload(relation string, conditions ...interface{}) common.SelectQuery {
	g.db = g.db.Preload(relation, conditions...)
	return g
}

func (g *GormSelectQuery) Order(order string) common.SelectQuery {
	g.db = g.db.Order(order)
	return g
}

func (g *GormSelectQuery) Limit(n int) common.SelectQuery {
	g.db = g.db.Limit(n)
	return g
}

func (g *GormSelectQuery) Offset(n int) common.SelectQuery {
	g.db = g.db.Offset(n)
	return g
}

func (g *GormSelectQuery) Group(group string) common.SelectQuery {
	g.db = g.db.Group(group)
	return g
}

func (g *GormSelectQuery) Having(having string, args ...interface{}) common.SelectQuery {
	g.db = g.db.Having(having, args...)
	return g
}

func (g *GormSelectQuery) Scan(ctx context.Context, dest interface{}) error {
	return g.db.WithContext(ctx).Find(dest).Error
}

func (g *GormSelectQuery) Count(ctx context.Context) (int, error) {
	var count int64
	err := g.db.WithContext(ctx).Count(&count).Error
	return int(count), err
}

func (g *GormSelectQuery) Exists(ctx context.Context) (bool, error) {
	var count int64
	err := g.db.WithContext(ctx).Limit(1).Count(&count).Error
	return count > 0, err
}

// GormInsertQuery implements InsertQuery for GORM
type GormInsertQuery struct {
	db     *gorm.DB
	model  interface{}
	values map[string]interface{}
}

func (g *GormInsertQuery) Model(model interface{}) common.InsertQuery {
	g.model = model
	g.db = g.db.Model(model)
	return g
}

func (g *GormInsertQuery) Table(table string) common.InsertQuery {
	g.db = g.db.Table(table)
	return g
}

func (g *GormInsertQuery) Value(column string, value interface{}) common.InsertQuery {
	if g.values == nil {
		g.values = make(map[string]interface{})
	}
	g.values[column] = value
	return g
}

func (g *GormInsertQuery) OnConflict(action string) common.InsertQuery {
	// GORM handles conflicts differently, this would need specific implementation
	return g
}

func (g *GormInsertQuery) Returning(columns ...string) common.InsertQuery {
	// GORM doesn't have explicit RETURNING, but updates the model
	return g
}

func (g *GormInsertQuery) Exec(ctx context.Context) (common.Result, error) {
	var result *gorm.DB
	if g.model != nil {
		result = g.db.WithContext(ctx).Create(g.model)
	} else if g.values != nil {
		result = g.db.WithContext(ctx).Create(g.values)
	} else {
		result = g.db.WithContext(ctx).Create(map[string]interface{}{})
	}
	return &GormResult{result: result}, result.Error
}

// GormUpdateQuery implements UpdateQuery for GORM
type GormUpdateQuery struct {
	db      *gorm.DB
	model   interface{}
	updates interface{}
}

func (g *GormUpdateQuery) Model(model interface{}) common.UpdateQuery {
	g.model = model
	g.db = g.db.Model(model)
	return g
}

func (g *GormUpdateQuery) Table(table string) common.UpdateQuery {
	g.db = g.db.Table(table)
	return g
}

func (g *GormUpdateQuery) Set(column string, value interface{}) common.UpdateQuery {
	if g.updates == nil {
		g.updates = make(map[string]interface{})
	}
	if updates, ok := g.updates.(map[string]interface{}); ok {
		updates[column] = value
	}
	return g
}

func (g *GormUpdateQuery) SetMap(values map[string]interface{}) common.UpdateQuery {
	g.updates = values
	return g
}

func (g *GormUpdateQuery) Where(query string, args ...interface{}) common.UpdateQuery {
	g.db = g.db.Where(query, args...)
	return g
}

func (g *GormUpdateQuery) Returning(columns ...string) common.UpdateQuery {
	// GORM doesn't have explicit RETURNING
	return g
}

func (g *GormUpdateQuery) Exec(ctx context.Context) (common.Result, error) {
	result := g.db.WithContext(ctx).Updates(g.updates)
	return &GormResult{result: result}, result.Error
}

// GormDeleteQuery implements DeleteQuery for GORM
type GormDeleteQuery struct {
	db    *gorm.DB
	model interface{}
}

func (g *GormDeleteQuery) Model(model interface{}) common.DeleteQuery {
	g.model = model
	g.db = g.db.Model(model)
	return g
}

func (g *GormDeleteQuery) Table(table string) common.DeleteQuery {
	g.db = g.db.Table(table)
	return g
}

func (g *GormDeleteQuery) Where(query string, args ...interface{}) common.DeleteQuery {
	g.db = g.db.Where(query, args...)
	return g
}

func (g *GormDeleteQuery) Exec(ctx context.Context) (common.Result, error) {
	result := g.db.WithContext(ctx).Delete(g.model)
	return &GormResult{result: result}, result.Error
}

// GormResult implements Result for GORM
type GormResult struct {
	result *gorm.DB
}

func (g *GormResult) RowsAffected() int64 {
	return g.result.RowsAffected
}

func (g *GormResult) LastInsertId() (int64, error) {
	// GORM doesn't directly provide last insert ID, would need specific implementation
	return 0, nil
}
