package playlist

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vale-tudo-devs/tvbarrapesada/remotecontrol/pkg/models"
)

const (
	cacheDir  = "/data"
	cacheFile = "playlist.m3u"
)

type Playlist struct {
	Items []PlaylistItem
}

type PlaylistItem struct {
	Name     string
	Category string
	URL      string
}

func UpdatePlaylist(ctx context.Context, playlistUrl, prefix string) {
	// Download playlist
	filePath := filepath.Join(cacheDir, fmt.Sprintf("%s-%s", prefix, cacheFile))
	file, err := downloadPlaylist(playlistUrl, filePath)
	if err != nil {
		log.Fatal("Failed to download playlist:", err)
	}

	// Parse playlist
	playlist, err := parsePlaylist(file)
	if err != nil {
		log.Fatal("Failed to parse playlist:", err)
	}
	log.Printf("Playlist parsed successfully: %d items", len(playlist.Items))

	s, err := models.NewAuthenticatedRedisClient(ctx)
	if err != nil {
		log.Fatal("Failed to create Redis client:", err)
	}
	s.Prefix = "channel"
	// Reset the channels in Redis
	err = s.DeleteAll(ctx)
	if err != nil {
		log.Fatal("Failed to reset channels in Redis:", err)
	}

	for i, item := range playlist.Items {
		if item.Name == "" || item.URL == "" {
			log.Printf("Skipping invalid item %d: missing name or URL", i)
			continue
		}
		// Save item to Redis
		s.Save(ctx, models.TvChannel{
			ID:       strconv.Itoa(i),
			Name:     item.Name,
			Category: item.Category,
			URL:      item.URL,
		})
	}
}

func parsePlaylist(filePath string) (*Playlist, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	playlist := &Playlist{Items: make([]PlaylistItem, 0)}
	scanner := bufio.NewScanner(file)

	var currentItem PlaylistItem
	categoryRegex := regexp.MustCompile(`group-title="([^"]*)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "#EXTM3U" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			// Extract category
			if matches := categoryRegex.FindStringSubmatch(line); len(matches) > 1 {
				currentItem.Category = matches[1]
			}
			// Extract name
			parts := strings.Split(line, ",")
			if len(parts) > 1 {
				currentItem.Name = parts[1]
			}
		} else if !strings.HasPrefix(line, "#") {
			currentItem.URL = line
			playlist.Items = append(playlist.Items, currentItem)
			currentItem = PlaylistItem{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return playlist, nil
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Creating directory: %s", dir)
		return os.MkdirAll(dir, 0755)
	}
	log.Printf("Directory already exists: %s", dir)
	return nil
}

func downloadPlaylist(url, filePath string) (string, error) {
	if err := ensureDir(cacheDir); err != nil {
		log.Fatal("Failed to create cache directory:", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Cache file not found, downloading from environment variable")

		resp, err := http.Get(url)
		if err != nil {
			log.Fatal("Failed to download playlist:", err)
		}
		defer resp.Body.Close()
		log.Printf("Successfully downloaded playlist")

		out, err := os.Create(filePath)
		if err != nil {
			log.Fatal("Failed to create cache file:", err)
		}
		defer out.Close()

		bytes, err := io.Copy(out, resp.Body)
		if err != nil {
			log.Fatal("Failed to write playlist to cache:", err)
		}
		log.Printf("Successfully wrote %d bytes to cache file", bytes)
	} else {
		log.Printf("Using existing cache file: %s", filePath)
	}

	return filePath, nil
}
