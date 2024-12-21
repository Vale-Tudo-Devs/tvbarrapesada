package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

const (
	remoteControlChannel = "tvbarrapesada"
)

type ChannelCommand struct {
	Command string `json:"command"`
	Tittle  string `json:"title"`
	URL     string `json:"url"`
}

func (s *RedisStore) Play(ctx context.Context, tvChannel TvChannel) error {
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
	return s.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}

func (s *RedisStore) Stop(ctx context.Context) error {
	command := ChannelCommand{
		Command: "stop",
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	log.Printf("Sending command: %+v", jsonData)
	return s.Client.Publish(ctx, remoteControlChannel, jsonData).Err()
}
