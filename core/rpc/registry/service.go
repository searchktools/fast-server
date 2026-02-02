package registry

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrMethodNotFound  = errors.New("method not found")
	ErrInvalidMethod   = errors.New("invalid method signature")
)

// ServiceRegistry manages registered services and methods
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*Service
}

// Service represents a registered service
type Service struct {
	Name    string
	Type    reflect.Type
	Value   reflect.Value
	Methods map[string]*Method
}

// Method represents a service method
type Method struct {
	Name      string
	Func      reflect.Value
	ArgType   reflect.Type
	ReplyType reflect.Type
}

// NewRegistry creates a new service registry
func NewRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*Service),
	}
}

// Register registers a service
// The service must have exported methods with signature:
//   func (s *Service) MethodName(ctx context.Context, arg *ArgType) (*ReplyType, error)
func (r *ServiceRegistry) Register(serviceName string, service interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc := &Service{
		Name:    serviceName,
		Type:    reflect.TypeOf(service),
		Value:   reflect.ValueOf(service),
		Methods: make(map[string]*Method),
	}

	// Scan for exported methods
	numMethods := svc.Type.NumMethod()
	for i := 0; i < numMethods; i++ {
		method := svc.Type.Method(i)
		mtype := method.Type

		// Method must be exported
		if method.PkgPath != "" {
			continue
		}

		// Check signature: func (receiver, context.Context, *arg) (*reply, error)
		if mtype.NumIn() != 3 || mtype.NumOut() != 2 {
			continue
		}

		// First arg must be context.Context
		ctxType := mtype.In(1)
		if !ctxType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			continue
		}

		// Second arg must be pointer
		argType := mtype.In(2)
		if argType.Kind() != reflect.Ptr {
			continue
		}

		// First return must be pointer
		replyType := mtype.Out(0)
		if replyType.Kind() != reflect.Ptr {
			continue
		}

		// Second return must be error
		errorType := mtype.Out(1)
		if !errorType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}

		svc.Methods[method.Name] = &Method{
			Name:      method.Name,
			Func:      method.Func,
			ArgType:   argType.Elem(),
			ReplyType: replyType.Elem(),
		}
	}

	r.services[serviceName] = svc
	return nil
}

// GetService returns a registered service
func (r *ServiceRegistry) GetService(name string) (*Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.services[name]
	if !ok {
		return nil, ErrServiceNotFound
	}
	return svc, nil
}

// GetMethod returns a method from a service
func (r *ServiceRegistry) GetMethod(serviceName, methodName string) (*Service, *Method, error) {
	svc, err := r.GetService(serviceName)
	if err != nil {
		return nil, nil, err
	}

	method, ok := svc.Methods[methodName]
	if !ok {
		return nil, nil, ErrMethodNotFound
	}

	return svc, method, nil
}

// Call invokes a registered method
func (r *ServiceRegistry) Call(ctx context.Context, serviceName, methodName string, arg interface{}) (interface{}, error) {
	svc, method, err := r.GetMethod(serviceName, methodName)
	if err != nil {
		return nil, err
	}

	// Prepare arguments
	argVal := reflect.ValueOf(arg)
	if argVal.Type() != reflect.PtrTo(method.ArgType) {
		return nil, fmt.Errorf("invalid argument type: expected %v, got %v",
			reflect.PtrTo(method.ArgType), argVal.Type())
	}

	ctxVal := reflect.ValueOf(ctx)

	// Call method
	returnValues := method.Func.Call([]reflect.Value{svc.Value, ctxVal, argVal})

	// Extract results
	reply := returnValues[0].Interface()
	errVal := returnValues[1]

	if !errVal.IsNil() {
		return nil, errVal.Interface().(error)
	}

	return reply, nil
}

// ListServices returns all registered service names
func (r *ServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	return names
}

// ListMethods returns all methods for a service
func (r *ServiceRegistry) ListMethods(serviceName string) ([]string, error) {
	svc, err := r.GetService(serviceName)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(svc.Methods))
	for name := range svc.Methods {
		names = append(names, name)
	}
	return names, nil
}
