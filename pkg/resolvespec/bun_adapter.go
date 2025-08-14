package resolvespec

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
)

// BunAdapter adapts Bun to work with our Database interface
// This demonstrates how the abstraction works with different ORMs
type BunAdapter struct {
	db *bun.DB
}

// NewBunAdapter creates a new Bun adapter
func NewBunAdapter(db *bun.DB) *BunAdapter {
	return &BunAdapter{db: db}
}

func (b *BunAdapter) NewSelect() SelectQuery {
	return &BunSelectQuery{query: b.db.NewSelect()}
}

func (b *BunAdapter) NewInsert() InsertQuery {
	return &BunInsertQuery{query: b.db.NewInsert()}
}

func (b *BunAdapter) NewUpdate() UpdateQuery {
	return &BunUpdateQuery{query: b.db.NewUpdate()}
}

func (b *BunAdapter) NewDelete() DeleteQuery {
	return &BunDeleteQuery{query: b.db.NewDelete()}
}

func (b *BunAdapter) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := b.db.ExecContext(ctx, query, args...)
	return &BunResult{result: result}, err
}

func (b *BunAdapter) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return b.db.NewRaw(query, args...).Scan(ctx, dest)
}

func (b *BunAdapter) BeginTx(ctx context.Context) (Database, error) {
	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	// For Bun, we'll return a special wrapper that holds the transaction
	return &BunTxAdapter{tx: tx}, nil
}

func (b *BunAdapter) CommitTx(ctx context.Context) error {
	// For Bun, we need to handle this differently
	// This is a simplified implementation
	return nil
}

func (b *BunAdapter) RollbackTx(ctx context.Context) error {
	// For Bun, we need to handle this differently  
	// This is a simplified implementation
	return nil
}

func (b *BunAdapter) RunInTransaction(ctx context.Context, fn func(Database) error) error {
	return b.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Create adapter with transaction  
		adapter := &BunTxAdapter{tx: tx}
		return fn(adapter)
	})
}

// BunSelectQuery implements SelectQuery for Bun
type BunSelectQuery struct {
	query *bun.SelectQuery
}

func (b *BunSelectQuery) Model(model interface{}) SelectQuery {
	b.query = b.query.Model(model)
	return b
}

func (b *BunSelectQuery) Table(table string) SelectQuery {
	b.query = b.query.Table(table)
	return b
}

func (b *BunSelectQuery) Column(columns ...string) SelectQuery {
	b.query = b.query.Column(columns...)
	return b
}

func (b *BunSelectQuery) Where(query string, args ...interface{}) SelectQuery {
	b.query = b.query.Where(query, args...)
	return b
}

func (b *BunSelectQuery) WhereOr(query string, args ...interface{}) SelectQuery {
	b.query = b.query.WhereOr(query, args...)
	return b
}

func (b *BunSelectQuery) Join(query string, args ...interface{}) SelectQuery {
	b.query = b.query.Join(query, args...)
	return b
}

func (b *BunSelectQuery) LeftJoin(query string, args ...interface{}) SelectQuery {
	b.query = b.query.Join("LEFT JOIN " + query, args...)
	return b
}

func (b *BunSelectQuery) Order(order string) SelectQuery {
	b.query = b.query.Order(order)
	return b
}

func (b *BunSelectQuery) Limit(n int) SelectQuery {
	b.query = b.query.Limit(n)
	return b
}

func (b *BunSelectQuery) Offset(n int) SelectQuery {
	b.query = b.query.Offset(n)
	return b
}

func (b *BunSelectQuery) Group(group string) SelectQuery {
	b.query = b.query.Group(group)
	return b
}

func (b *BunSelectQuery) Having(having string, args ...interface{}) SelectQuery {
	b.query = b.query.Having(having, args...)
	return b
}

func (b *BunSelectQuery) Scan(ctx context.Context, dest interface{}) error {
	return b.query.Scan(ctx, dest)
}

func (b *BunSelectQuery) Count(ctx context.Context) (int, error) {
	count, err := b.query.Count(ctx)
	return count, err
}

func (b *BunSelectQuery) Exists(ctx context.Context) (bool, error) {
	return b.query.Exists(ctx)
}

// BunInsertQuery implements InsertQuery for Bun
type BunInsertQuery struct {
	query  *bun.InsertQuery
	values map[string]interface{}
}

func (b *BunInsertQuery) Model(model interface{}) InsertQuery {
	b.query = b.query.Model(model)
	return b
}

func (b *BunInsertQuery) Table(table string) InsertQuery {
	b.query = b.query.Table(table)
	return b
}

func (b *BunInsertQuery) Value(column string, value interface{}) InsertQuery {
	if b.values == nil {
		b.values = make(map[string]interface{})
	}
	b.values[column] = value
	return b
}

func (b *BunInsertQuery) OnConflict(action string) InsertQuery {
	b.query = b.query.On(action)
	return b
}

func (b *BunInsertQuery) Returning(columns ...string) InsertQuery {
	if len(columns) > 0 {
		b.query = b.query.Returning(columns[0])
	}
	return b
}

func (b *BunInsertQuery) Exec(ctx context.Context) (Result, error) {
	if b.values != nil {
		// For Bun, we need to handle this differently
		for k, v := range b.values {
			b.query = b.query.Set("? = ?", bun.Ident(k), v)
		}
	}
	result, err := b.query.Exec(ctx)
	return &BunResult{result: result}, err
}

// BunUpdateQuery implements UpdateQuery for Bun
type BunUpdateQuery struct {
	query *bun.UpdateQuery
}

func (b *BunUpdateQuery) Model(model interface{}) UpdateQuery {
	b.query = b.query.Model(model)
	return b
}

func (b *BunUpdateQuery) Table(table string) UpdateQuery {
	b.query = b.query.Table(table)
	return b
}

func (b *BunUpdateQuery) Set(column string, value interface{}) UpdateQuery {
	b.query = b.query.Set(column+" = ?", value)
	return b
}

func (b *BunUpdateQuery) SetMap(values map[string]interface{}) UpdateQuery {
	for column, value := range values {
		b.query = b.query.Set(column+" = ?", value)
	}
	return b
}

func (b *BunUpdateQuery) Where(query string, args ...interface{}) UpdateQuery {
	b.query = b.query.Where(query, args...)
	return b
}

func (b *BunUpdateQuery) Returning(columns ...string) UpdateQuery {
	if len(columns) > 0 {
		b.query = b.query.Returning(columns[0])
	}
	return b
}

func (b *BunUpdateQuery) Exec(ctx context.Context) (Result, error) {
	result, err := b.query.Exec(ctx)
	return &BunResult{result: result}, err
}

// BunDeleteQuery implements DeleteQuery for Bun
type BunDeleteQuery struct {
	query *bun.DeleteQuery
}

func (b *BunDeleteQuery) Model(model interface{}) DeleteQuery {
	b.query = b.query.Model(model)
	return b
}

func (b *BunDeleteQuery) Table(table string) DeleteQuery {
	b.query = b.query.Table(table)
	return b
}

func (b *BunDeleteQuery) Where(query string, args ...interface{}) DeleteQuery {
	b.query = b.query.Where(query, args...)
	return b
}

func (b *BunDeleteQuery) Exec(ctx context.Context) (Result, error) {
	result, err := b.query.Exec(ctx)
	return &BunResult{result: result}, err
}

// BunResult implements Result for Bun
type BunResult struct {
	result sql.Result
}

func (b *BunResult) RowsAffected() int64 {
	if b.result == nil {
		return 0
	}
	rows, _ := b.result.RowsAffected()
	return rows
}

func (b *BunResult) LastInsertId() (int64, error) {
	if b.result == nil {
		return 0, nil
	}
	return b.result.LastInsertId()
}

// BunTxAdapter wraps a Bun transaction to implement the Database interface
type BunTxAdapter struct {
	tx bun.Tx
}

func (b *BunTxAdapter) NewSelect() SelectQuery {
	return &BunSelectQuery{query: b.tx.NewSelect()}
}

func (b *BunTxAdapter) NewInsert() InsertQuery {
	return &BunInsertQuery{query: b.tx.NewInsert()}
}

func (b *BunTxAdapter) NewUpdate() UpdateQuery {
	return &BunUpdateQuery{query: b.tx.NewUpdate()}
}

func (b *BunTxAdapter) NewDelete() DeleteQuery {
	return &BunDeleteQuery{query: b.tx.NewDelete()}
}

func (b *BunTxAdapter) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := b.tx.ExecContext(ctx, query, args...)
	return &BunResult{result: result}, err
}

func (b *BunTxAdapter) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return b.tx.NewRaw(query, args...).Scan(ctx, dest)
}

func (b *BunTxAdapter) BeginTx(ctx context.Context) (Database, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (b *BunTxAdapter) CommitTx(ctx context.Context) error {
	return b.tx.Commit()
}

func (b *BunTxAdapter) RollbackTx(ctx context.Context) error {
	return b.tx.Rollback()
}

func (b *BunTxAdapter) RunInTransaction(ctx context.Context, fn func(Database) error) error {
	return fn(b) // Already in transaction
}