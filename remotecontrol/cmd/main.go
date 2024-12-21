package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/bot"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/models"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/playlist"
)

func main() {
	ctx := context.Background()
	bot, err := bot.New()
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SKIP_CHANNEL_DB_UPDATE") == "" {
		playlist.UpdatePlaylist(ctx)
	}

	// Start redis sub for local dev

	if os.Getenv("REDIS_LOCAL_DEV") != "" {
		log.Printf("Starting local dev redis sub")
		r, err := models.NewAuthenticatedRedisClient(ctx)
		if err != nil {
			log.Fatal(err)
		}
		go r.StartDevSub(ctx)
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
