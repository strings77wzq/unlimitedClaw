package tools_test

import (
	"fmt"

	"github.com/strings77wzq/golem/core/tools"
)

func ExampleNewRegistry() {
	registry := tools.NewRegistry()

	tool := &tools.MockTool{
		ToolName:        "echo",
		ToolDescription: "echoes input",
	}

	registry.Register(tool)

	retrieved, ok := registry.Get("echo")
	if ok {
		fmt.Println(retrieved.Name())
	}
	// Output: echo
}

func ExampleRegistry_ListDefinitions() {
	registry := tools.NewRegistry()

	tool1 := &tools.MockTool{
		ToolName:        "echo",
		ToolDescription: "echoes input",
	}
	tool2 := &tools.MockTool{
		ToolName:        "calc",
		ToolDescription: "performs calculation",
	}

	registry.Register(tool1)
	registry.Register(tool2)

	definitions := registry.ListDefinitions()
	for _, def := range definitions {
		fmt.Println(def.Name)
	}
	// Output:
	// calc
	// echo
}
