package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kkdai/youtube/v2"
)

const (
	remoteControlChannel = "tvbarrapesada"
)

type ChannelCommand struct {
	Command string `json:"command"`
	Tittle  string `json:"title"`
	URL     string `json:"url"`
}

func (r *RedisStore) Play(ctx context.Context, id int64) error {
	// Stop any previous channel and wait one second
	r.Stop(ctx)
	time.Sleep(2 * time.Second)
	r.Prefix = "channel"
	tvChannel, err := r.GetChannelByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get channel by id: %w", err)
	}

	command := ChannelCommand{
		Command: "play",
		Tittle:  tvChannel.Name,
		URL:     tvChannel.URL,
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	log.Printf("Sending command: %+v", jsonData)
	return r.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}

func (r *RedisStore) Stop(ctx context.Context) error {
	r.Prefix = "channel"
	command := ChannelCommand{
		Command: "stop",
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	log.Printf("Sending command: %+v", jsonData)
	return r.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}

func (r *RedisStore) Restart(ctx context.Context) error {
	r.Prefix = "channel"
	command := ChannelCommand{
		Command: "restart",
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	log.Printf("Sending command: %+v", jsonData)
	return r.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}

func (r *RedisStore) RandomChannel(ctx context.Context) (*TvChannel, error) {
	randChannel, err := r.GetRandomChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get random channel: %w", err)
	}

	channel, err := r.GetChannelByID(ctx, randChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel by id: %w", err)
	}

	return channel, r.Play(ctx, randChannel)
}

func (r *RedisStore) PlayYoutube(ctx context.Context, url string) error {
	r.Prefix = "channel"
	videoTitle, err := getYoutubeTitle(url)
	if err != nil {
		log.Printf("failed to get youtube title: %v", err)
		videoTitle = "Youtube Video"
	}
	command := ChannelCommand{
		Command: "play",
		Tittle:  videoTitle,
		URL:     url,
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	log.Printf("Sending command: %+v", jsonData)
	return r.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}

func getYoutubeTitle(url string) (string, error) {
	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		return "", fmt.Errorf("failed to get video info: %w", err)
	}

	return video.Title, nil
}
