package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager manages application configuration
type Manager struct {
	values map[string]interface{}
	mu     sync.RWMutex
	
	// Watchers for configuration changes
	watchers map[string][]func(string, interface{})
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		values:   make(map[string]interface{}),
		watchers: make(map[string][]func(string, interface{})),
	}
}

// Set sets a configuration value
func (m *Manager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.values[key] = value
	
	// Notify watchers
	if watchers, exists := m.watchers[key]; exists {
		for _, watcher := range watchers {
			go watcher(key, value)
		}
	}
}

// Get gets a configuration value
func (m *Manager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	value, exists := m.values[key]
	return value, exists
}

// GetString gets a string configuration value
func (m *Manager) GetString(key string, defaultValue ...string) string {
	if value, exists := m.Get(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetInt gets an integer configuration value
func (m *Manager) GetInt(key string, defaultValue ...int) int {
	if value, exists := m.Get(key); exists {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetBool gets a boolean configuration value
func (m *Manager) GetBool(key string, defaultValue ...bool) bool {
	if value, exists := m.Get(key); exists {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "yes" || v == "1"
		case int:
			return v != 0
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetFloat gets a float configuration value
func (m *Manager) GetFloat(key string, defaultValue ...float64) float64 {
	if value, exists := m.Get(key); exists {
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0.0
}

// GetDuration gets a duration configuration value
func (m *Manager) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	if value, exists := m.Get(key); exists {
		switch v := value.(type) {
		case time.Duration:
			return v
		case string:
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		case int64:
			return time.Duration(v)
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetStringSlice gets a string slice configuration value
func (m *Manager) GetStringSlice(key string, defaultValue ...[]string) []string {
	if value, exists := m.Get(key); exists {
		switch v := value.(type) {
		case []string:
			return v
		case []interface{}:
			result := make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					result[i] = str
				}
			}
			return result
		case string:
			// Parse comma-separated string
			return strings.Split(v, ",")
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return []string{}
}

// Watch watches for configuration changes
func (m *Manager) Watch(key string, callback func(string, interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.watchers[key] = append(m.watchers[key], callback)
}

// LoadFromEnv loads configuration from environment variables
func (m *Manager) LoadFromEnv(prefix string) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := parts[0]
		value := parts[1]
		
		// Check if key has the prefix
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		
		// Remove prefix
		if prefix != "" {
			key = strings.TrimPrefix(key, prefix)
			key = strings.TrimPrefix(key, "_")
		}
		
		// Convert key to lowercase and replace underscores with dots
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", ".")
		
		m.Set(key, value)
	}
}

// LoadFromJSON loads configuration from JSON file
func (m *Manager) LoadFromJSON(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}
	
	m.loadFromMap("", values)
	return nil
}

// loadFromMap recursively loads configuration from a map
func (m *Manager) loadFromMap(prefix string, values map[string]interface{}) {
	for key, value := range values {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		
		// If value is a map, recurse
		if nested, ok := value.(map[string]interface{}); ok {
			m.loadFromMap(fullKey, nested)
		} else {
			m.Set(fullKey, value)
		}
	}
}

// SaveToJSON saves configuration to JSON file
func (m *Manager) SaveToJSON(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, err := json.MarshalIndent(m.values, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// Unmarshal unmarshals configuration into a struct
func (m *Manager) Unmarshal(prefix string, target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Get target value and type
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	
	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}
	
	targetType := targetValue.Type()
	
	// Iterate through struct fields
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := targetValue.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		// Get config key from tag or field name
		configKey := field.Tag.Get("config")
		if configKey == "" {
			configKey = strings.ToLower(field.Name)
		}
		
		// Add prefix
		if prefix != "" {
			configKey = prefix + "." + configKey
		}
		
		// Get value from config
		value, exists := m.values[configKey]
		if !exists {
			continue
		}
		
		// Set field value
		if err := m.setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}
	
	return nil
}

// setFieldValue sets a reflect.Value from an interface{} value
func (m *Manager) setFieldValue(field reflect.Value, value interface{}) error {
	valueReflect := reflect.ValueOf(value)
	
	// Handle type conversion
	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
		} else {
			field.SetString(fmt.Sprintf("%v", value))
		}
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case float64:
			field.SetInt(int64(v))
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				field.SetInt(i)
			}
		}
		
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case string:
			field.SetBool(v == "true" || v == "yes" || v == "1")
		case int:
			field.SetBool(v != 0)
		}
		
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case float32:
			field.SetFloat(float64(v))
		case int:
			field.SetFloat(float64(v))
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				field.SetFloat(f)
			}
		}
		
	case reflect.Slice:
		if valueReflect.Kind() == reflect.Slice {
			field.Set(valueReflect)
		}
		
	default:
		if valueReflect.Type().ConvertibleTo(field.Type()) {
			field.Set(valueReflect.Convert(field.Type()))
		} else {
			return fmt.Errorf("cannot convert %v to %v", valueReflect.Type(), field.Type())
		}
	}
	
	return nil
}

// GetAll returns all configuration values
func (m *Manager) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		result[k] = v
	}
	
	return result
}

// Delete deletes a configuration value
func (m *Manager) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.values, key)
}

// Clear clears all configuration values
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.values = make(map[string]interface{})
}
