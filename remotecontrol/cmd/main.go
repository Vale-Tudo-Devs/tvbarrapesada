package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/bot"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/models"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/playlist"
)

func main() {
	ctx := context.Background()
	b, err := bot.New()
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SKIP_CHANNEL_DB_UPDATE") == "" {
		log.Printf("Updating TV playlist")
		playlist.UpdatePlaylist(ctx, os.Getenv("TV_PLAYLIST_URL"), "channel")
		log.Printf("Updating Movies playlist")
		playlist.UpdatePlaylist(ctx, os.Getenv("MOVIES_PLAYLIST_URL"), "movies")
	}

	err = b.DiscordSession.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	bot.AddCommands(b.DiscordSession)
	log.Println("Discord Bot is now running.")

	// Make channel to keep bot running and handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to check viewer status every minute
	go func() {
		ticker := time.NewTicker(120 * time.Second)
		defer ticker.Stop()

		r, err := models.NewAuthenticatedRedisClient(ctx)
		if err != nil {
			log.Panicf("Error creating redis client: %v\n", err)
		}

		for {
			select {
			case <-ticker.C:
				isWatching := bot.IsAnyoneWatching(ctx, b.DiscordSession)
				if !isWatching {
					log.Println("No one is watching, stopping bot.")
					r.Stop(ctx)
				}
			case <-stop:
				return
			}
		}
	}()

	// Wait for signal to terminate
	<-stop
	log.Println("Gracefully shutting down...")

	err = b.DiscordSession.Close()
	if err != nil {
		log.Printf("Error closing Discord session: %v\n", err)
	}

}
