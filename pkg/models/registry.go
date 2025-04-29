package models

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	modelRegistry      = make(map[string]interface{})
	functionRegistry   = make(map[string]interface{})
	modelRegistryMutex sync.RWMutex
	funcRegistryMutex  sync.RWMutex
)

// RegisterModel registers a model type with the registry
// The model must be a struct or a pointer to a struct
// e.g RegisterModel(&ModelPublicUser{},"public.user")
func RegisterModel(model interface{}, name string) error {
	modelRegistryMutex.Lock()
	defer modelRegistryMutex.Unlock()

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	if name == "" {
		name = modelType.Name()
	}
	modelRegistry[name] = model
	return nil
}

// RegisterFunction register a function with the registry
func RegisterFunction(fn interface{}, name string) {
	funcRegistryMutex.Lock()
	defer funcRegistryMutex.Unlock()
	functionRegistry[name] = fn
}

// GetModelByName retrieves a model from the registry by its type name
func GetModelByName(name string) (interface{}, error) {
	modelRegistryMutex.RLock()
	defer modelRegistryMutex.RUnlock()

	if modelRegistry[name] == nil {
		return nil, fmt.Errorf("model not found: %s", name)
	}
	return modelRegistry[name], nil
}

// IterateModels iterates over all models in the registry
func IterateModels(fn func(name string, model interface{})) {
	modelRegistryMutex.RLock()
	defer modelRegistryMutex.RUnlock()

	for name, model := range modelRegistry {
		fn(name, model)
	}
}

// GetModels returns a list of all models in the registry
func GetModels() []interface{} {
	models := make([]interface{}, 0)
	modelRegistryMutex.RLock()
	defer modelRegistryMutex.RUnlock()
	for _, model := range modelRegistry {
		models = append(models, model)
	}
	return models
}
