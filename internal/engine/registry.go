package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Registry manages all registered commands.
type Registry struct {
	commands map[string]*Command
	mu       sync.RWMutex
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*Command),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd *Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Name] = cmd
}

// Get retrieves a command by name.
func (r *Registry) Get(name string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[name]
	return cmd, ok
}

// All returns all registered commands.
func (r *Registry) All() []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmds := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// Names returns all registered command names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// Execute runs a command by name with the given parameters.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (Result, error) {
	cmd, ok := r.Get(name)
	if !ok {
		return Result{}, fmt.Errorf("command not found: %s", name)
	}

	// Apply defaults for missing parameters
	if params == nil {
		params = make(map[string]any)
	}
	for _, p := range cmd.Parameters {
		if _, exists := params[p.Name]; !exists && p.Default != nil {
			params[p.Name] = p.Default
		}
	}

	// Validate required parameters
	for _, p := range cmd.Parameters {
		if p.Required {
			if _, exists := params[p.Name]; !exists {
				return Result{}, fmt.Errorf("missing required parameter: %s", p.Name)
			}
		}
	}

	start := time.Now()
	result, err := cmd.Execute(ctx, params)
	result.Duration = time.Since(start)

	return result, err
}

// Count returns the number of registered commands.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.commands)
}
