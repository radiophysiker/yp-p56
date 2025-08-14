package main

import (
	"log"
	
	"github.com/radiophysiker/d56/internal/app"
)

func main() {
	err := app.Run()
	if err != nil {
		log.Fatalf("cannot run the app! %v", err)
	}
}
