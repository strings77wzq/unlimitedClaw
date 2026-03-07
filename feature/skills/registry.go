package skills

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe skill registry.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*Skill
}

// NewRegistry creates a new skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*Skill),
	}
}

// Register registers a skill. Returns an error if a skill with the same name already exists.
func (r *Registry) Register(skill *Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	if skill.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("skill with name %q already registered", skill.Name)
	}

	r.skills[skill.Name] = skill
	return nil
}

// Get retrieves a skill by name. Returns the skill and true if found, nil and false otherwise.
func (r *Registry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, ok := r.skills[name]
	return skill, ok
}

// List returns all registered skills sorted by name.
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// Remove removes a skill by name. Returns true if the skill existed and was removed.
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return false
	}

	delete(r.skills, name)
	return true
}
