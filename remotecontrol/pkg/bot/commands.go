package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/models"
)

func AddCommands(s *discordgo.Session) {
	ctx := context.Background()
	r, err := models.NewAuthenticatedRedisClient(ctx)
	if err != nil {
		log.Printf("Error creating redis client: %v\n", err)
		return
	}

	// Define and create the TV command
	r.Prefix = "channel"
	channelsLen, err := r.GetChannelCounter(ctx)
	if err != nil {
		log.Printf("Error getting channel count: %v\n", err)
		return
	}
	channelsLen-- // The counter starts at 0

	tvCommand := &discordgo.ApplicationCommand{
		Name:        "tv",
		Description: "Set the TV channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "channel",
				Description: fmt.Sprintf("Channel ID (0-%d)", channelsLen),
				Required:    true,
				MinValue:    &[]float64{0}[0],
				MaxValue:    float64(channelsLen),
			},
		},
	}

	c, err := s.ApplicationCommandCreate(s.State.User.ID, "", tvCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("tv command added: %v\n", c.Name)

	youtubeCommand := &discordgo.ApplicationCommand{
		Name:        "yt",
		Description: "Play a Youtube video",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "A Youtube video URL",
				Required:    true,
			},
		},
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", youtubeCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("yt command added: %v\n", c.Name)

	// Define and create the Stop command
	stopCommand := &discordgo.ApplicationCommand{
		Name:        "stop",
		Description: "Stop the TV",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", stopCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("stop command added: %v\n", c.Name)

	// Define and create the Search command
	searchCommand := &discordgo.ApplicationCommand{
		Name:        "search",
		Description: "Search for a TV channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "query",
				Description: "Search for a channel, you can use multiple words",
				Required:    true,
			},
		},
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", searchCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("search command added: %v\n", c.Name)

	// Define and create restart command
	restartCommand := &discordgo.ApplicationCommand{
		Name:        "restart",
		Description: "Restart the bot",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", restartCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("restart command added: %v\n", c.Name)

	randomCommand := &discordgo.ApplicationCommand{
		Name:        "random",
		Description: "Set a random TV channel",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", randomCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("random command added: %v\n", c.Name)

	catalogCommand := &discordgo.ApplicationCommand{
		Name:        "catalog",
		Description: "Download a CSV with all TV channels",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", catalogCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("catalog command added: %v\n", c.Name)
}
