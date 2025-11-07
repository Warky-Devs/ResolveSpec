package modelregistry

import (
	"fmt"
	"reflect"
	"sync"
)

// DefaultModelRegistry implements ModelRegistry interface
type DefaultModelRegistry struct {
	models map[string]interface{}
	mutex  sync.RWMutex
}

// Global default registry instance
var defaultRegistry = &DefaultModelRegistry{
	models: make(map[string]interface{}),
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *DefaultModelRegistry {
	return &DefaultModelRegistry{
		models: make(map[string]interface{}),
	}
}

func (r *DefaultModelRegistry) RegisterModel(name string, model interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.models[name]; exists {
		return fmt.Errorf("model %s already registered", name)
	}

	// Validate that model is a non-pointer struct
	modelType := reflect.TypeOf(model)
	if modelType == nil {
		return fmt.Errorf("model cannot be nil")
	}

	originalType := modelType

	// Unwrap pointers, slices, and arrays to check the underlying type
	for modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array {
		modelType = modelType.Elem()
	}

	// Validate that the underlying type is a struct
	if modelType.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct or pointer to struct, got %s", originalType.String())
	}

	// If a pointer/slice/array was passed, unwrap to the base struct
	if originalType != modelType {
		// Create a zero value of the struct type
		model = reflect.New(modelType).Elem().Interface()
	}

	// Additional check: ensure model is not a pointer
	finalType := reflect.TypeOf(model)
	if finalType.Kind() == reflect.Ptr {
		return fmt.Errorf("model must be a non-pointer struct, got pointer to %s. Use MyModel{} instead of &MyModel{}", finalType.Elem().Name())
	}

	r.models[name] = model
	return nil
}

func (r *DefaultModelRegistry) GetModel(name string) (interface{}, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	model, exists := r.models[name]
	if !exists {
		return nil, fmt.Errorf("model %s not found", name)
	}
	
	return model, nil
}

func (r *DefaultModelRegistry) GetAllModels() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range r.models {
		result[k] = v
	}
	return result
}

func (r *DefaultModelRegistry) GetModelByEntity(schema, entity string) (interface{}, error) {
	// Try full name first
	fullName := fmt.Sprintf("%s.%s", schema, entity)
	if model, err := r.GetModel(fullName); err == nil {
		return model, nil
	}

	// Fallback to entity name only
	return r.GetModel(entity)
}

// Global convenience functions using the default registry

// RegisterModel registers a model with the default global registry
func RegisterModel(model interface{}, name string) error {
	return defaultRegistry.RegisterModel(name, model)
}

// GetModelByName retrieves a model from the default global registry by name
func GetModelByName(name string) (interface{}, error) {
	return defaultRegistry.GetModel(name)
}

// IterateModels iterates over all models in the default global registry
func IterateModels(fn func(name string, model interface{})) {
	defaultRegistry.mutex.RLock()
	defer defaultRegistry.mutex.RUnlock()

	for name, model := range defaultRegistry.models {
		fn(name, model)
	}
}

// GetModels returns a list of all models in the default global registry
func GetModels() []interface{} {
	defaultRegistry.mutex.RLock()
	defer defaultRegistry.mutex.RUnlock()

	models := make([]interface{}, 0, len(defaultRegistry.models))
	for _, model := range defaultRegistry.models {
		models = append(models, model)
	}
	return models
}