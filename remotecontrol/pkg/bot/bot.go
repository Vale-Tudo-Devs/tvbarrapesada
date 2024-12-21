package bot

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/models"
)

type Bot struct {
	DiscordSession *discordgo.Session
}

func New() (*Bot, error) {
	token, ok := os.LookupEnv("DISCORD_BOT_TOKEN")
	if !ok {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN environment variable is not set")
	}

	s, err := discordgo.New(fmt.Sprintf("Bot %s", token))
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	s.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentsGuildPresences |
		discordgo.IntentGuildMembers |
		discordgo.IntentGuildVoiceStates |
		discordgo.IntentMessageContent

	s.AddHandler(tvHandler)

	return &Bot{
		DiscordSession: s,
	}, nil
}

func tvHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	r, err := models.NewAuthenticatedRedisClient(ctx)
	if err != nil {
		log.Printf("Error creating redis client: %v\n", err)
		return
	}

	// Define and create the TV command
	tvCommand := &discordgo.ApplicationCommand{
		Name:        "tv",
		Description: "Set the TV channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "channel",
				Description: "Channel ID (0-20000)",
				Required:    false,
				MinValue:    &[]float64{0}[0],
				MaxValue:    20000,
			},
		},
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", tvCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}

	// Define and create the Stop command
	stopCommand := &discordgo.ApplicationCommand{
		Name:        "stop",
		Description: "Stop the TV",
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", stopCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}

	switch i.ApplicationCommandData().Name {
	case "tv":
		channelId := i.ApplicationCommandData().Options[0].IntValue()
		log.Printf("TV command received from user: %s - channelId: %d", i.Member.User.Username, channelId)

		err := r.Play(ctx, channelId)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}
		// Respond to the interaction
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("TV channel set to %d", i.ApplicationCommandData().Options[0].IntValue()),
			},
		})

		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
		}
	case "stop":
		log.Printf("Stop command received from user: %s", i.Member.User.Username)

		r.Stop(ctx)

		// Respond to the interaction
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "TV stopped",
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
		}
	default:
		log.Printf("Unknown command: %s\n", i.ApplicationCommandData().Name)
	}
}
