package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

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
	r.Prefix = "channel"

	switch i.ApplicationCommandData().Name {
	case "tv":
		channelId := i.ApplicationCommandData().Options[0].IntValue()
		log.Printf("TV command received from user: %s - channelId: %d", i.Member.User.Username, channelId)

		channelName, err := r.GetChannelByID(ctx, channelId)
		if err != nil {
			log.Printf("Error getting channel by ID: %v\n", err)
			return
		}

		err = r.Play(ctx, channelId)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}

		err = r.RegisterCurrentChannel(ctx, channelName)
		if err != nil {
			log.Printf("Error registering current channel: %v\n", err)
		}

		// Respond to the interaction
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("TV channel set to %d - %s", channelId, channelName.Name),
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
	case "search":
		query := i.ApplicationCommandData().Options[0].StringValue()
		log.Printf("Search command received from user: %s - query: %s", i.Member.User.Username, query)

		channels, err := r.SearchChannelsByName(ctx, query)
		if err != nil {
			log.Printf("Error searching for channel: %v\n", err)
			return
		}

		// Respond to the interaction
		var content string
		if len(channels) == 0 {
			content = "No channels found"
		} else {
			content = "Channels found:\n"
			for _, channel := range channels {
				content += fmt.Sprintf("%s - %s\n", channel.ID, channel.Name)
			}
		}
		// Limit content to 1980 characters
		truncatedMessage := "\n\nSearch truncated, be more specific"
		maxLen := 2000 - len(truncatedMessage)
		if len(content) > maxLen {

			content = fmt.Sprintf("%s%s", content[:maxLen], truncatedMessage)
		}
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
			return
		}

		log.Printf("Channels found: %v\n", channels)
	case "restart":
		log.Printf("Restart command received from user: %s", i.Member.User.Username)
		err := r.Restart(ctx)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}

		time.Sleep(2 * time.Second)
		currentChannel, err := r.GetCurrentChannel(ctx)
		if err != nil {
			log.Printf("Error getting current channel: %v\n", err)
		}

		channelID, err := strconv.ParseInt(currentChannel.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing channel ID: %v\n", err)
			return
		}
		err = r.Play(ctx, channelID)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}

		// Respond to the interaction
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Bot restarted",
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
		}
	case "random":
		log.Printf("Random command received from user: %s", i.Member.User.Username)
		channel, err := r.RandomChannel(ctx)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}

		// Respond to the interaction
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Random channel set to %s - %s", channel.ID, channel.Name),
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
		}
	case "yt":
		log.Printf("YT command received from user: %s", i.Member.User.Username)
		url := i.ApplicationCommandData().Options[0].StringValue()
		tittle, err := r.PlayYoutube(ctx, url)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
		}

		// Respond to the interaction
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Playing Youtube video: %s", tittle),
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v\n", err)
		}

	default:
		log.Printf("Unknown command: %s\n", i.ApplicationCommandData().Name)
	}
}

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
				Required:    false,
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
		Description: "Get a random TV channel",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", randomCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("random command added: %v\n", c.Name)

	youtubeCommand := &discordgo.ApplicationCommand{
		Name:        "yt",
		Description: "Set the TV channel to a youtube video",
	}

	c, err = s.ApplicationCommandCreate(s.State.User.ID, "", youtubeCommand)
	if err != nil {
		log.Printf("Error creating slash command: %v\n", err)
		return
	}
	log.Printf("yt command added: %v\n", c.Name)
}

func DeleteCommands(s *discordgo.Session) {
	commands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		log.Printf("Error getting slash commands: %v\n", err)
		return
	}

	for _, command := range commands {
		err = s.ApplicationCommandDelete(s.State.User.ID, "", command.ID)
		if err != nil {
			log.Printf("Error deleting slash command: %v\n", err)
		}
		log.Printf("Command deleted: %s\n", command.Name)
	}
}

func IsAnyoneWatching(ctx context.Context, s *discordgo.Session) bool {
	guilds, err := s.UserGuilds(200, "", "", true)
	if err != nil {
		return true
	}
	for _, guild := range guilds {
		// Register oncall users
		guildID := guild.ID
		members, err := s.GuildMembers(guildID, "", 1000)
		if err != nil {
			log.Printf("error fetching members for guild %s: %v", guildID, err)
			continue
		}
		oncallUsersCount := 0
		for _, member := range members {
			if member.User.Bot {
				continue
			}
			vs, _ := s.State.VoiceState(guildID, member.User.ID) // it errors out if the user is not in a voice channel, ignore it
			if vs != nil && vs.ChannelID != "" {
				// Check if user is on an ignored channel
				currentVoiceChannel, err := s.Channel(vs.ChannelID)
				if err != nil {
					log.Printf("error fetching channel for user %s: %v", member.User.ID, err)
					continue
				}
				ignoredChannels := strings.Split(os.Getenv("DISCORD_IGNORED_CHANNELS"), ",")
				if slices.Contains(ignoredChannels, currentVoiceChannel.Name) {
					log.Printf("Ignoring user %s in ignored channel %s", member.User.ID, currentVoiceChannel.Name)
					continue
				}

				oncallUsersCount++
			}
		}
		// If anyone other than the bot is watching, return true
		if oncallUsersCount > 1 {
			return true
		}
	}
	return false
}
