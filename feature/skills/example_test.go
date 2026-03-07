package skills_test

import (
	"fmt"

	"github.com/strings77wzq/golem/feature/skills"
)

func ExampleNewRegistry() {
	registry := skills.NewRegistry()

	skill := &skills.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Version:     "1.0.0",
	}

	registry.Register(skill)

	retrieved, ok := registry.Get("test-skill")
	if ok {
		fmt.Println(retrieved.Name)
	}
	// Output: test-skill
}

func ExampleRegistry_List() {
	registry := skills.NewRegistry()

	skill1 := &skills.Skill{
		Name:        "skill-alpha",
		Description: "First skill",
		Version:     "1.0.0",
	}
	skill2 := &skills.Skill{
		Name:        "skill-beta",
		Description: "Second skill",
		Version:     "2.0.0",
	}

	registry.Register(skill1)
	registry.Register(skill2)

	skillList := registry.List()
	for _, s := range skillList {
		fmt.Println(s.Name)
	}
	// Output:
	// skill-alpha
	// skill-beta
}
