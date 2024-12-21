package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/bot"
)

func main() {
	bot, err := bot.New()
	if err != nil {
		log.Fatal(err)
	}

	err = bot.DiscordSession.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}
	log.Println("Discord Bot is now running.")

	// Make channel to keep bot running and handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal to terminate
	<-stop
	log.Println("Gracefully shutting down...")

	err = bot.DiscordSession.Close()
	if err != nil {
		log.Printf("Error closing Discord session: %v\n", err)
	}
}
