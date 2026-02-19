package main

import (
	"errors"

	"github.com/Madlite/gator/internal/config"
)

type State struct {
	cfg *config.Config
}

type Command struct {
	name string
	args []string
}

type Commands struct {
	handler map[string]func(*State, Command) error
}

func (c *Commands) run(s *State, cmd Command) error {
	err := c.handler[cmd.name](s, cmd)
	if err != nil {
		return errors.New("running command failed")
	}
	return nil
}

func (c *Commands) register(name string, f func(*State, Command) error) {
	c.handler[name] = f
}
