package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	_ "github.com/mattn/go-sqlite3"
)

func playAudioFile(s *discordgo.Session, guildID string, channelID string, filePath string, memberToKick *discordgo.User) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96

	encodeSession, err := dca.EncodeFile(filePath, options)
	if err != nil {
		fmt.Println("Error encoding audio file: ", err)
		return
	}
	defer encodeSession.Cleanup()

	// Join the voice channel
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		fmt.Println("Error joining voice channel: ", err)
		return
	}
	vc.Speaking(true)
	defer vc.Disconnect()

	done := make(chan error)

	// Send the Opus data to Discord
	dca.NewStream(encodeSession, vc, done)

	// Wait for the playback to finish
	err = <-done
	if err != nil && err != io.EOF {
		fmt.Println("Error streaming audio to Discord: ", err)
		return
	}
	s.GuildMemberMove(guildID, memberToKick.ID, nil)

}

func searchUserByUsername(s *discordgo.Session, guildID, username string) (*discordgo.User, error) {
	members, err := s.GuildMembers(guildID, "", 1000)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		if strings.EqualFold(member.User.Username, username) {
			return member.User, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

func getSoundFilePath(filename string) string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory: ", err)
		return ""
	}

	return filepath.Join(cwd, "sounds", filename+".mp3")
}

func sendPrivateMessage(s *discordgo.Session, m *discordgo.MessageCreate, username, message string) {

	member, err := searchUserByUsername(s, m.GuildID, username)
	if err != nil {
		fmt.Printf("%s is not a user", username)
		return
	}

	channel, err := s.UserChannelCreate(member.ID)
	if err != nil {
		fmt.Println("Error creating DM channel:", err)
		return
	}

	s.ChannelMessageSend(channel.ID, message)
}

func openSQLDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "mafia.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func closeSQLDatabase(db *sql.DB) {
	if db != nil {
		err := db.Close()
		if err != nil {
			log.Println("Error closing database:", err)
		}
	}
}

func createGamesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mafiaGames (
			gameID INTEGER PRIMARY KEY AUTOINCREMENT,
			guildID TEXT NOT NULL,
			playerCount INTEGER,
			winner TEXT,
			status TEXT NOT NULL
		)
	`)
	return err
}

func createPlayersTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mafiaPlayers (
			playerID INTEGER PRIMARY KEY AUTOINCREMENT,
			gameID INTEGER NOT NULL,
			playerCount INTEGER,
			user TEXT NOT NULL,
			role TEXT,
			FOREIGN KEY (gameID) REFERENCES mafiaGames(gameID)
		)
	`)
	return err
}

func createTurnsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mafiaTurns (
			Turn INTEGER,
			gameID INTEGER NOT NULL,
			role TEXT,
			user TEXT NOT NULL,
			action TEXT,
			userActedUpon TEXT,
			FOREIGN KEY (gameID) REFERENCES mafiaGames(gameID)
			PRIMARY KEY (GameID, user, Turn)
		)
	`)
	return err
}

func InitializeSchema(db *sql.DB) {
	createGamesTable(db)
	createPlayersTable(db)
	createTurnsTable(db)
}
