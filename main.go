package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const (
	command_prefix = "!"
)

func main() {

	//Initialize Discord Session
	rangus, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Print("Error creating Discord session: ", err)
	}

	rangus.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	db, err := openSQLDatabase()
	if err != nil {
		fmt.Print("There was an error opening the SQL DB: ", err)
		return
	}
	InitializeSchema(db)
	defer closeSQLDatabase(db)

	fmt.Println("Rangus Bot Online. Listening for busters.")

	// Handlers
	messageHandler := WrapMessageCreate(db)

	// Add the closure as a handler
	rangus.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageHandler(s, m)
	})

	//Open websocket and begin listening
	err = rangus.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Waits for signal to stop
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	rangus.Close()
}

type MessageHandler func(s *discordgo.Session, m *discordgo.MessageCreate)

func WrapMessageCreate(db *sql.DB) MessageHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {

		// Ignore posts that come from the bot
		if m.Author.ID == s.State.User.ID {
			return
		}

		fmt.Printf("\nReceived command from %s in channel %s\n", m.Author.Username, m.ChannelID)
		fmt.Println("Heard message: " + m.Content)

		//Route cases to the appropriate handler function
		switch {
		case strings.Contains(strings.ToUpper(m.Content), "LETHAL COMPANY"):
			handleLethalCompany(s, m)

		case strings.HasPrefix(m.Content, command_prefix+"begone"):
			handleBegoneCommand(s, m)

		case strings.HasPrefix(m.Content, command_prefix+"game"):
			GameRouter(s, m, db)

		case strings.HasPrefix(m.Content, command_prefix+"action"):
			handleGameActions(s, m, db)

		default:
			return
		}
	}
}
