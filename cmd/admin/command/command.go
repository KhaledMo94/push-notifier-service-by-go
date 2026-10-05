package command

import (
	"context"
	"fmt"
	"sort"
)

type Command interface {
	Name() string
	Usage() string
	Run(ctx context.Context, args []string) error
}

var registry = map[string]Command{}

func Register(c Command) {
	if _, exists := registry[c.Name()]; exists {
		// in the begaining of app
		panic(fmt.Sprintf("command %q registered twice", c.Name()))
	}
	registry[c.Name()] = c
}

// All returns the registered commands sorted by name.
func All() []Command {
	commands := make([]Command, 0, len(registry))
	for _, c := range registry {
		commands = append(commands, c)
	}
	sort.Slice(commands, func(i, j int) bool { return commands[i].Name() < commands[j].Name() })
	return commands
}
