package extensions

import (
	"fmt"
	"sync"
)

// Container represents a Firefox-style container tab grouping.
// Each container provides isolated cookie storage, localStorage, and
// sessionStorage for tabs assigned to it.
type Container struct {
	// ID uniquely identifies the container.
	ID string `json:"id"`

	// Name is the human-readable container name (e.g. "Work", "Personal").
	Name string `json:"name"`

	// Color is the hex color used for UI indicators (tab badge, address bar).
	Color string `json:"color"`

	// Icon is an optional icon identifier for the container.
	Icon string `json:"icon,omitempty"`
}

// NewContainer creates a new container with the given name and color.
func NewContainer(id, name, color string) *Container {
	return &Container{
		ID:    id,
		Name:  name,
		Color: color,
	}
}

// ContainerManager manages the set of named containers. It is the source of
// truth for container state and supports CRUD operations.
//
// ContainerManager is not safe for concurrent use; drive it from the UI loop.
type ContainerManager struct {
	mu        sync.Mutex
	containers map[string]*Container
	nextID    int
}

// NewContainerManager creates an empty ContainerManager with the default
// (non-container) context pre-registered as an implicit container.
func NewContainerManager() *ContainerManager {
	return &ContainerManager{
		containers: make(map[string]*Container),
		nextID:     1,
	}
}

// Create adds a new container with the given name and color. It returns the
// created container, or an error if a container with the same name already
// exists.
func (m *ContainerManager) Create(name, color string) (*Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.containers {
		if c.Name == name {
			return nil, fmt.Errorf("container %q already exists", name)
		}
	}

	id := fmt.Sprintf("c%d", m.nextID)
	m.nextID++
	c := NewContainer(id, name, color)
	m.containers[id] = c
	return c, nil
}

// Get returns the container with the given ID, or nil if not found.
func (m *ContainerManager) Get(id string) *Container {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.containers[id]
}

// GetByName returns the container with the given name, or nil if not found.
func (m *ContainerManager) GetByName(name string) *Container {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.containers {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Update modifies the name and/or color of the container with the given ID.
// Empty strings leave the field unchanged. It returns the updated container,
// or an error if the container is not found.
func (m *ContainerManager) Update(id, name, color string) (*Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.containers[id]
	if !ok {
		return nil, fmt.Errorf("container %q not found", id)
	}
	if name != "" {
		c.Name = name
	}
	if color != "" {
		c.Color = color
	}
	return c, nil
}

// Remove deletes the container with the given ID. It returns true if the
// container existed and was removed.
func (m *ContainerManager) Remove(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.containers[id]; !ok {
		return false
	}
	delete(m.containers, id)
	return true
}

// List returns a snapshot of all containers in insertion order.
func (m *ContainerManager) List() []*Container {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*Container, 0, len(m.containers))
	for _, c := range m.containers {
		out = append(out, c)
	}
	return out
}

// Count returns the number of containers.
func (m *ContainerManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.containers)
}

// Has returns true if a container with the given ID exists.
func (m *ContainerManager) Has(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.containers[id]
	return ok
}

// DefaultContainerID returns the sentinel ID used for the default
// (non-container) browsing context.
const DefaultContainerID = ""
