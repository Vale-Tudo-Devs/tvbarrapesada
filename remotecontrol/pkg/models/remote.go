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

func (r *RedisStore) Play(ctx context.Context, id int64) error {
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

func (r *RedisStore) StartDevSub(ctx context.Context) {
	pubsub := r.Client.PSubscribe(ctx, fmt.Sprintf("%s:*", remoteControlChannel))
	defer pubsub.Close()

	// Listen on channel
	ch := pubsub.Channel()
	for msg := range ch {
		log.Printf("Received message: %s", msg.Payload)
	}
}
