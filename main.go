package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Madlite/gator/internal/config"
	"github.com/Madlite/gator/internal/database"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

func main() {
	config, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("Error with config: %v", err)
	}

	db, err := sql.Open("postgres", config.DbUrl)
	if err != nil {
		log.Fatal("Unable to connect to database")
	}
	dbQueries := database.New(db)

	state := &State{
		cfg: &config,
		db:  dbQueries,
	}
	commands := Commands{
		handler: make(map[string]func(*State, Command) error),
	}
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)

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
	userName := cmd.args[0]
	_, err := s.db.GetUser(context.Background(), userName)
	if err != nil {
		log.Printf("User does not exist in database")
		os.Exit(1)
	}

	err = s.cfg.SetUser(userName)
	if err != nil {
		return errors.New("Errow with login setuser")
	}

	fmt.Println("User has been set")
	return nil
}

func handlerRegister(s *State, cmd Command) error {
	if len(cmd.args) != 1 {
		return errors.New("The login handler expects a single argument, the username.")
	}

	userDB := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	}

	user, err := s.db.CreateUser(context.Background(), userDB)
	if err != nil {
		os.Exit(1)
	}
	s.cfg.SetUser(user.Name)
	fmt.Printf("Current user set to %v", user.Name)
	log.Println(user)
	return nil
}
