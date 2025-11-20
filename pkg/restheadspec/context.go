package restheadspec

import (
	"context"
)

// Context keys for request-scoped data
type contextKey string

const (
	contextKeySchema    contextKey = "schema"
	contextKeyEntity    contextKey = "entity"
	contextKeyTableName contextKey = "tableName"
	contextKeyModel     contextKey = "model"
	contextKeyModelPtr  contextKey = "modelPtr"
	contextKeyOptions   contextKey = "options"
)

// WithSchema adds schema to context
func WithSchema(ctx context.Context, schema string) context.Context {
	return context.WithValue(ctx, contextKeySchema, schema)
}

// GetSchema retrieves schema from context
func GetSchema(ctx context.Context) string {
	if v := ctx.Value(contextKeySchema); v != nil {
		return v.(string)
	}
	return ""
}

// WithEntity adds entity to context
func WithEntity(ctx context.Context, entity string) context.Context {
	return context.WithValue(ctx, contextKeyEntity, entity)
}

// GetEntity retrieves entity from context
func GetEntity(ctx context.Context) string {
	if v := ctx.Value(contextKeyEntity); v != nil {
		return v.(string)
	}
	return ""
}

// WithTableName adds table name to context
func WithTableName(ctx context.Context, tableName string) context.Context {
	return context.WithValue(ctx, contextKeyTableName, tableName)
}

// GetTableName retrieves table name from context
func GetTableName(ctx context.Context) string {
	if v := ctx.Value(contextKeyTableName); v != nil {
		return v.(string)
	}
	return ""
}

// WithModel adds model to context
func WithModel(ctx context.Context, model interface{}) context.Context {
	return context.WithValue(ctx, contextKeyModel, model)
}

// GetModel retrieves model from context
func GetModel(ctx context.Context) interface{} {
	return ctx.Value(contextKeyModel)
}

// WithModelPtr adds model pointer to context
func WithModelPtr(ctx context.Context, modelPtr interface{}) context.Context {
	return context.WithValue(ctx, contextKeyModelPtr, modelPtr)
}

// GetModelPtr retrieves model pointer from context
func GetModelPtr(ctx context.Context) interface{} {
	return ctx.Value(contextKeyModelPtr)
}

// WithOptions adds request options to context
func WithOptions(ctx context.Context, options ExtendedRequestOptions) context.Context {
	return context.WithValue(ctx, contextKeyOptions, options)
}

// GetOptions retrieves request options from context
func GetOptions(ctx context.Context) *ExtendedRequestOptions {
	if v := ctx.Value(contextKeyOptions); v != nil {
		if opts, ok := v.(ExtendedRequestOptions); ok {
			return &opts
		}
	}
	return nil
}

// WithRequestData adds all request-scoped data to context at once
func WithRequestData(ctx context.Context, schema, entity, tableName string, model, modelPtr interface{}, options ExtendedRequestOptions) context.Context {
	ctx = WithSchema(ctx, schema)
	ctx = WithEntity(ctx, entity)
	ctx = WithTableName(ctx, tableName)
	ctx = WithModel(ctx, model)
	ctx = WithModelPtr(ctx, modelPtr)
	ctx = WithOptions(ctx, options)
	return ctx
}
