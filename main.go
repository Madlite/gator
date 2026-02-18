package main

import (
	"fmt"

	"github.com/Madlite/gator/internal/config"
)

func main() {
	fmt.Println("Hello, world!")
	config.ReadConfig()
}
