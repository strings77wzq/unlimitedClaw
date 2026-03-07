package bus_test

import (
	"fmt"

	"github.com/strings77wzq/golem/core/bus"
)

func ExampleNew() {
	b := bus.New()
	defer b.Close()

	ch := b.Subscribe("test")

	b.Publish("test", "hello world")

	msg := <-ch
	fmt.Println(msg.(string))
	// Output: hello world
}
