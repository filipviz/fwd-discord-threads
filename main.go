package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var config Config

type Config struct {
	Sources      []SourceChannels `json:"sources"`
	Destinations []DestChannels   `json:"destinations"`
}

type SourceChannels struct {
	GuildID    string   `json:"guildId"`
	ChannelIDs []string `json:"channels"`
}

type DestChannels struct {
	GuildID  string        `json:"guildId"`
	Channels []DestChannel `json:"channels"`
}

type DestChannel struct {
	ChannelID string   `json:"channelId"`
	VoterIDs  []string `json:"voterIds"`
}

func main() {
	// Open and read config and .env
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Printf("Could not open config.json: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Printf("Could not decode config.json: %v\n", err)
		os.Exit(1)
	}

	err = godotenv.Load(".env")
	if err != nil {
		fmt.Printf("Could not load .env: %v\n", err)
		os.Exit(1)
	}

	discordToken, foundToken := os.LookupEnv("DISCORD_TOKEN")
	if !foundToken || discordToken == "" {
		fmt.Printf("Could not find DISCORD_TOKEN in .env. Exiting.\n")
		os.Exit(1)
	}

	// Setup Discord client and add handlers
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Printf("Could not create Discord client: %v\n", err)
		os.Exit(1)
	}
	dg.AddHandler(handleConnect)
	dg.AddHandler(handleThreadCreate)

	if err := dg.Open(); err != nil {
		fmt.Printf("Error opening connection: %v\n", err)
		return
	}
	defer dg.Close()
	fmt.Println("Discord connection opened.")

	// Wait for signal to stop
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV, syscall.SIGHUP)
	<-sc
}

func handleConnect(s *discordgo.Session, m *discordgo.Connect) {
	fmt.Printf("Connected to %s\n", "session")
}

func handleThreadCreate(s *discordgo.Session, m *discordgo.ThreadCreate) {
	for _, source := range config.Sources {
		if m.GuildID == source.GuildID {
			for _, channelId := range source.ChannelIDs {
				if channelId == m.ParentID {
					msg := fmt.Sprintf("New thread: *[%s](https://discord.com/channels/%s/%s/%s)*", m.Name, m.GuildID, m.Channel.ID, m.ID)
					notifyDestinations(s, msg)
				}
			}
		}
	}
}

func notifyDestinations(s *discordgo.Session, msg string) {
	for _, destGuild := range config.Destinations {
		for _, destChannel := range destGuild.Channels {
			tags := make([]string, len(destChannel.VoterIDs))
			for _, voterId := range destChannel.VoterIDs {
				tags = append(tags, fmt.Sprintf("<@%s>", voterId))
			}
			msg = msg + " " + strings.Join(tags, "")

			_, err := s.ChannelMessageSend(destChannel.ChannelID, msg)
			if err != nil {
				fmt.Printf("Could not message channel %v: %v\n", destChannel.ChannelID, err)
			}
		}
	}
}