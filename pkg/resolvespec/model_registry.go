package resolvespec

import (
	"fmt"
	"sync"
)

// DefaultModelRegistry implements ModelRegistry interface
type DefaultModelRegistry struct {
	models map[string]interface{}
	mutex  sync.RWMutex
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