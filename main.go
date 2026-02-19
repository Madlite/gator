package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Madlite/gator/internal/config"
)

func main() {
	config, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("Error with config: %v", err)
	}
	state := &State{
		cfg: &config,
	}
	commands := Commands{
		handler: make(map[string]func(*State, Command) error),
	}
	commands.register("login", handlerLogin)

	input := os.Args
	if len(input) < 3 {
		os.Exit(1)
	}

	cmd, args := input[1], input[2:]
	command := Command{
		name: cmd,
		args: args,
	}
	err = commands.run(state, command)

}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.args) != 1 {
		return errors.New("The login handler expects a single argument, the username.")
	}

	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return errors.New("Errow with login setuser")
	}

	fmt.Println("User has been set")
	return nil
}
