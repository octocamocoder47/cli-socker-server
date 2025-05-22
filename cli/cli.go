package cli

import (
	"fmt"
	"os"
	"strings"

	prompt "github.com/c-bata/go-prompt"
)

var commands = map[string]func(args []string){
	"list": func(args []string) {
		fmt.Println("Connected Clients:")
		for id, conn := range CLIENTS {
			fmt.Printf("ID: %d, Addr: %s\n", id, conn.String())
		}
	},
	"broadcast": func(args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: broadcast <message>")
			return
		}
		msg := strings.Join(args, " ")
		server.BroadCast([]byte(msg))
	},
	"exit": func(args []string) {
		fmt.Println("Shutting down.")
		os.Exit(0)
	},
	"help": func(args []string) {
		fmt.Println("Available commands:")
		for cmd := range commands {
			fmt.Println(" -", cmd)
		}
	},
}

// Executor runs when a user presses Enter
func Executor(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]

	if action, ok := commands[cmd]; ok {
		action(args)
	} else {
		fmt.Println("Unknown command. Type 'help'.")
	}
}

// Completer enables tab-completion
func Completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{}
	for name := range commands {
		s = append(s, prompt.Suggest{Text: name})
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
