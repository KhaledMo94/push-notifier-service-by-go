package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KhaledMo94/push-notifier-service-by-go/cmd/admin/command"
)

func main() {
	commands := command.All()

	if len(os.Args) < 2 {
		printUsage(commands)
		os.Exit(2)
	}

	cmd := find(commands, os.Args[1])
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage(commands)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args[2:]); err != nil {
		stop()
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func find(commands []command.Command, name string) command.Command {
	for _, c := range commands {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func printUsage(commands []command.Command) {
	fmt.Fprintln(os.Stderr, "usage: admin <command> [flags]")
	fmt.Fprintln(os.Stderr, "\ncommands:")
	for _, c := range commands {
		fmt.Fprintf(os.Stderr, "  %-14s %s\n", c.Name(), c.Usage())
	}
	fmt.Fprintln(os.Stderr, "\nrun 'admin <command> -h' for a command's flags")
}
