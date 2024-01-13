package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func newDayMessages(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {
	serverTownChannel, err := getTownChatID(s, m)
	if err != nil {
		fmt.Println(err)
	}

	turnDay := getTurnDay(s, m, db)
	newDayMessage := "##  🌄  ~Dawn of Day " + turnDay + " ~ 🌄  "

	currentgameLobby, err := getmafiaGameLobby(db, m.GuildID, "Active")
	if err != nil {
		fmt.Println("There was an error getting the game lobby: ", err)
	}

	s.ChannelMessageSend(serverTownChannel, newDayMessage)

	newDayPrivateMessage := "## ~ New Day Action ~ \nWhat action to take today, onii-chan??"
	for currentgameLobby.Next() {
		var usersCol string
		currentgameLobby.Scan(&usersCol)
		sendPrivateMessage(s, m, usersCol, newDayPrivateMessage)
	}
}

func getTownChatID(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {

	serverGuildChannels, err := s.GuildChannels(m.GuildID)
	if err != nil {
		fmt.Println("There was an error getting the townchat ID: ", err)
		return "", nil
	}

	for _, channel := range serverGuildChannels {
		if channel.Name == townChannel {
			return channel.ID, nil
		}
	}
	return "", nil
}

func getTurnDay(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) string {
	return "3"
}

func handleGameActions(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {
	args := strings.Split(m.Content, " ")

	userTargetInput := args[1]
	targetUser, err := guildMemberLookup(s, m, userTargetInput, db)
	if err != nil {
		sendPrivateMessage(s, m, m.Author.Username, "Trouble performing this action, please try again!")
	}

	filePlayerTurnAction(s, m, db, m.Author.Username, targetUser)

}

func guildMemberLookup(s *discordgo.Session, m *discordgo.MessageCreate, userToLookUp string, db *sql.DB) (string, error) {
	users, err := s.GuildMembersSearch(m.GuildID, userToLookUp, 1000)
	if err != nil {
		sendPrivateMessage(s, m, m.Author.Username, "Trouble performing this action, please try again!")
		return "", err
	}

	for _, guildMember := range users {
		if strings.EqualFold(guildMember.User.Username, userToLookUp) {
			if userIsInCurrentGame(s, m, m.GuildID, db, guildMember.User.Username) {
				return guildMember.User.Username, nil
			}
		}
	}
	return "", nil
}

func filePlayerTurnAction(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB, player, target string) {
	fmt.Print("Hi!")
}

func userIsInCurrentGame(s *discordgo.Session, m *discordgo.MessageCreate, guildID string, db *sql.DB, guildMember string) bool {
	users, err := getmafiaGameLobby(db, guildID, "Active")
	if err != nil {
		fmt.Println("There was an issue getting the active game in : ", m.GuildID)
	}

	for users.Next() {
		var usersCol string
		users.Scan(&usersCol)
		if usersCol == guildMember {
			return true
		}
	}
	return false
}

func populateTurnTable(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB, turn int) {
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Error beginning transaction: ", err)
		return
	}
	defer tx.Rollback()

	users, err := getmafiaGameLobby(db, m.GuildID, "Active")
	if err != nil {
		fmt.Println("There was an issue getting the active game in : ", m.GuildID)
	}
	gameID, err := ExistingGameWithStatus(db, m.GuildID, "Active")
	if err != nil {
		fmt.Print("There was an error finding an existing game with status Active in " + m.GuildID)
	}

	for users.Next() {
		var usersCol string
		users.Scan(&usersCol)
		userRole := mafiaGetPlayerRole(s, m, db, usersCol, gameID)
		_, err := tx.Exec(
			"INSERT INTO mafiaTurns(Turn, gameID, role, user) VALUES (?,?,?,?)", turn, gameID, userRole, m.Author.Username)
		if err != nil {
			fmt.Printf("Failed to insert %s into the game lobby=: %s", m.Author.Username, err)
		}
	}
}

func mafiaGetPlayerRole(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB, user string, gameID int) string {
	var role string
	err := db.QueryRow("SELECT role FROM mafiaPlayers WHERE gameID = ? AND user = ?", gameID, user).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}
