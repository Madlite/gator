package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
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
	commands.register("reset", handlerReset)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAggregator)

	input := os.Args
	if len(input) < 2 {
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

func handlerReset(s *State, cmd Command) error {
	if len(cmd.args) > 0 {
		return errors.New("reset takes no args")
	}

	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return errors.New("Unable to reset users database")
	}

	log.Println("Reset user database")
	return nil
}

func handlerUsers(s *State, cmd Command) error {
	if len(cmd.args) > 0 {
		return errors.New("users takes no args")
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return errors.New("Unable to fetch users from database")
	}

	for _, user := range users {
		output := "* " + user.Name
		if s.cfg.CurrentUserName == user.Name {
			output += " (current)"
		}
		fmt.Println(output)
	}

	return nil
}

func handlerAggregator(s *State, cmd Command) error {
	feedUrl := "https://www.wagslane.dev/index.xml"
	rssFeed, err := fetchFeed(context.Background(), feedUrl)
	if err != nil {
		return errors.New("Error getting example feed")
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}
	fmt.Println(rssFeed)
	return nil
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	client := http.DefaultClient
	payload := RSSFeed{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return &payload, errors.New("Could not create request")
	}
	req.Header.Set("User-Agent", "gator")

	resp, err := client.Do(req)
	if err != nil {
		return &payload, errors.New("Error creating client response")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &payload, errors.New("Body errors")
	}

	var data RSSFeed
	if err := xml.Unmarshal(body, &data); err != nil {
		return &payload, errors.New("Xml errors")
	}
	return &data, nil
}
