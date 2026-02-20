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
	commands.register("users", handlerGetUsers)
	commands.register("agg", handlerAggregator)
	commands.register("addfeed", handlerAddFeed)
	commands.register("feeds", handlerGetFeeds)

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
	if err != nil {
		log.Fatal(err)
	}

}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.args) != 1 {
		log.Println("The login handler expects a single argument, the username")
		os.Exit(1)
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
		log.Println("The register handler expects a single argument, the username")
		os.Exit(1)
	}
	userName := cmd.args[0]
	_, err := s.db.GetUser(context.Background(), userName)
	if err == nil {
		log.Println("User already exist in database")
		os.Exit(1)
	}

	userDB := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      userName,
	}

	user, err := s.db.CreateUser(context.Background(), userDB)
	if err != nil {
		log.Println(err)
		return errors.New("Errors creating user in database")
	}
	s.cfg.SetUser(user.Name)
	fmt.Printf("Current user set to %v", user.Name)
	log.Println(user)
	return nil
}

func handlerReset(s *State, cmd Command) error {
	if len(cmd.args) > 0 {
		log.Println("reset takes no args")
		os.Exit(1)
	}

	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return errors.New("Unable to reset users database")
	}

	log.Println("Reset user database")
	return nil
}

func handlerGetUsers(s *State, cmd Command) error {
	if len(cmd.args) > 0 {
		log.Println("users takes no args")
		os.Exit(1)
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
	if len(cmd.args) > 0 {
		log.Println("Agg takes no args")
		os.Exit(1)
	}

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

func handlerAddFeed(s *State, cmd Command) error {
	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}
	if len(cmd.args) != 2 {
		log.Println("addfeed takes 2 args, username and url")
		os.Exit(1)
	}

	userName, url := cmd.args[0], cmd.args[1]
	dbParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      userName,
		Url:       url,
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), dbParams)
	if err != nil {
		return errors.New("Error creating feed entry in database")
	}

	fmt.Printf("* ID:            %s\n", feed.ID)
	fmt.Printf("* Created:       %v\n", feed.CreatedAt)
	fmt.Printf("* Updated:       %v\n", feed.UpdatedAt)
	fmt.Printf("* Name:          %s\n", feed.Name)
	fmt.Printf("* URL:           %s\n", feed.Url)
	fmt.Printf("* UserID:        %s\n", feed.UserID)

	return nil
}

func handlerGetFeeds(s *State, cmd Command) error {
	if len(cmd.args) > 0 {
		log.Println("feeds takes no args")
		os.Exit(1)
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return errors.New("Error fetching feeds in database")
	}
	for _, feed := range feeds {
		fmt.Println(feed.FeedName)
		fmt.Println(feed.FeedUrl)
		fmt.Println(feed.UserName)
	}

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
