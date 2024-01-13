package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RoleID       = "1193379628736335872" //Dead Role for the Server (Will need to automatically get this later!!!)
	townChannel  = "town-chat"
	deadChannel  = "graveyard"
	mafiaChannel = "baddies"
)

var Emojis = [...]string{
	":fearful:", ":cold_sweat:", ":scream:", ":clown:", ":smiling_imp:", ":imp:", ":japanese_ogre:",
	":japanese_goblin:", ":skull:", ":skull_crossbones:", ":ghost:", ":alien:", ":space_invader:",
	":robot:", ":levitate:", ":green_heart:", ":black_heart:", ":unicorn:", ":bat:", ":owl:", ":spider:",
	":spider_web:", ":wilted_rose:", ":chocolate_bar:", ":candy:", ":lollipop:", ":house_abandoned:",
	":night_with_stars:", ":flying_saucer:", ":full_moon:", ":new_moon_with_face:", ":cloud_lightning:",
	":jack_o_lantern:", ":crystal_ball:", ":performing_arts:", ":candle:", ":dagger:", ":chains:", ":coffin:",
	":urn:",
}

type CharacterCounts struct {
	Good    int
	Bad     int
	Neutral int
}

func GameRouter(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {

	args := strings.Split(m.Content, " ")

	switch args[1] {

	//Game Setup and Info
	case "join":
		handleJoinLobbyCommand(s, m, db)

	case "lobby":
		handleActiveLobbyCommand(s, m, db)

	case "start":
		handlemafiaGameStartCommand(s, m, db)

	//Player Information
	case "role":
		handleMyRoleCommand(s, m)

	//Game Functionality
	case "vote":
		sendPrivateMessage(s, m, m.Author.Username, "Vote yourself bitch ass n'wah")
	case "kill":
		killPlayer(s, m, m.GuildID, m.Author.ID)

	}

}

func handleMyRoleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	//role := constructPlayerRole()
	//sendPrivateMessage(s, m, m.Author.Username, messageToSend)
}

func killPlayer(s *discordgo.Session, m *discordgo.MessageCreate, guildID, username string) {
	params := &discordgo.GuildMemberParams{
		Roles: &[]string{RoleID},
	}

	_, err := s.GuildMemberEdit(guildID, username, params)
	if err != nil {
		fmt.Println("Error assigning role:", err)
		return
	}
}

func handleJoinLobbyCommand(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {

	ExistingLobby, err := ExistingGameWithStatus(db, m.GuildID, "Lobby")
	if err != nil {
		fmt.Println("There was a problem finding any lobbies: ", err)
	}

	if ExistingLobby == 1 {
		createNewLobby(db, m, 1)
		addPlayerToExistingLobby(db, m, ExistingLobby)
	} else {
		addPlayerToExistingLobby(db, m, ExistingLobby)
	}

	handleActiveLobbyCommand(s, m, db)

}

func ExistingGameWithStatus(db *sql.DB, guildID string, status string) (int, error) {
	var gameID int
	err := db.QueryRow("SELECT gameID FROM mafiaGames WHERE Status = ? AND guildID = ?", status, guildID).Scan(&gameID)
	if err == sql.ErrNoRows {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return gameID, nil

}

func UserInExistingGameLobby(db *sql.DB, m *discordgo.MessageCreate) (int, error) {
	var gameID int
	err := db.QueryRow("SELECT gameID FROM mafiaGames WHERE Status = 'Lobby' AND guildID = ? AND user = ?", m.GuildID, m.Author.Username).Scan(&gameID)
	if err != nil {
		return 0, err
	}
	return gameID, nil

}

func createNewLobby(db *sql.DB, m *discordgo.MessageCreate, gameID int) {
	_, err := db.Exec("INSERT INTO mafiaGames (gameID, guildID,playerCount,status) VALUES (?,?,?,?)", gameID, m.GuildID, 0, "Lobby")
	if err != nil {
		fmt.Println("There was an error creating a new lobby: ", err)
	}
}

func addPlayerToExistingLobby(db *sql.DB, m *discordgo.MessageCreate, ExistingLobby int) {
	_, err := db.Exec("INSERT INTO mafiaPlayers (playerID, gameID,user) VALUES (?,?,?)", m.Author.ID, ExistingLobby, m.Author.Username)
	if err != nil {
		fmt.Printf("Failed to insert %s into the game lobby=: %s", m.Author.Username, err)
	}
	_, updateErr := db.Exec("UPDATE mafiaGames SET playerCount = playerCount + 1 WHERE gameID = ?", ExistingLobby)
	if updateErr != nil {
		fmt.Print("There was an issue updating playercount:", updateErr)
	}
	_, updatePlayersErr := db.Exec("UPDATE mafiaPlayers SET playerCount = playerCount + 1 WHERE gameID = ?", ExistingLobby)
	if updatePlayersErr != nil {
		fmt.Print("There was an issue updating playercount:", updatePlayersErr)
	}
}

func handleActiveLobbyCommand(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {
	randomEmoji := rand.Intn(len(Emojis))
	message := "## " + Emojis[randomEmoji] + " ~Current Active Lobby~ " + Emojis[randomEmoji]

	rows, err := getmafiaGameLobby(db, m.GuildID, "Lobby")
	if err != nil {
		fmt.Print("There was an error gathering players from the current lobby: ", err)
	}

	index := 1
	for rows.Next() {
		var usersCol string
		rows.Scan(&usersCol)

		discordUserNickName, err := getActiveNickname(s, m, m.GuildID, usersCol)
		if err != nil {
			fmt.Print("There was an issue while looking up the nickname :", err)
		}
		message = fmt.Sprintf("%s\n%s %s", message, "> ", discordUserNickName)
		index++
	}
	s.ChannelMessageSend(m.ChannelID, message)
}

func getActiveNickname(s *discordgo.Session, m *discordgo.MessageCreate, guildID, username string) (string, error) {
	members, err := s.GuildMembersSearch(guildID, username, 1000)
	if err != nil {
		return "", err
	}

	for _, member := range members {
		if strings.EqualFold(member.User.Username, username) || strings.EqualFold(member.Nick, username) {
			if member.Nick != "" {
				return member.Nick, nil
			}

			return member.User.GlobalName, nil
		}
	}

	return "", fmt.Errorf("member with username %s not found in guild %s", username, guildID)
}

func getmafiaGameLobby(db *sql.DB, guildID, status string) (*sql.Rows, error) {
	rows, err := db.Query("SELECT user FROM mafiaPLayers INNER JOIN mafiaGames on mafiaPlayers.GameID = mafiaGames.GameID WHERE mafiaGames.status = ? AND guildID = ?", status, guildID)
	if err != nil {
		return rows, err
	}
	return rows, err
}

func handlemafiaGameStartCommand(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB) {
	rows, err := getmafiaGameLobby(db, m.GuildID, "Lobby")
	if err != nil {
		fmt.Println("There was an issue getting the game lobby:", err)
	}

	var playersSlice []string

	for rows.Next() {
		var usersCol string
		rows.Scan(&usersCol)
		playersSlice = append(playersSlice, usersCol)
	}
	updateGameStatus(0, db, m.GuildID, "Active")
	assignRoles(s, m, playersSlice, db)
	createGameChannels(s, m)
	populateTurnTable(s, m, db, 1)
	newDayMessages(s, m, db)
}

func updateGameStatus(gameID int, db *sql.DB, guildID string, status string) {
	var err error
	//If game Lobby is not specified, grab the existing game lobby
	if gameID == 0 {
		gameID, err = ExistingGameWithStatus(db, guildID, status)
		if err != nil {
			fmt.Print("There was an error getting the existing game Lobby with status: ", status)
		}
	}
	_, updateErr := db.Exec("UPDATE mafiaGames SET Status = ?  WHERE gameID = ?", status, gameID)
	if updateErr != nil {
		fmt.Print("There was an issue updating the game status:", err)
	}
}

func assignRoles(s *discordgo.Session, m *discordgo.MessageCreate, Users []string, db *sql.DB) {
	var counts CharacterCounts
	groupsize := len(Users)

	//  Character Counts Table
	switch {
	case groupsize < 6:
		counts.Bad = 1
		counts.Good = groupsize - counts.Bad
		fmt.Printf("Game starting with %d players", groupsize)

	case groupsize == 6 || groupsize == 7:
		counts.Bad = 2
		counts.Good = groupsize - counts.Bad
		fmt.Printf("Game starting with %d players", groupsize)

	case groupsize >= 8:
		counts.Bad = 2
		counts.Neutral = 1
		counts.Good = groupsize - counts.Bad - counts.Good
		fmt.Printf("Game starting with %d players", groupsize)

	}
	//Shuffle the Users array for random roles
	rand.NewSource(time.Now().UnixNano())

	rand.Shuffle(len(Users), func(i, j int) {
		Users[i], Users[j] = Users[j], Users[i]
	})

	for _, username := range Users {
		if counts.Neutral > 0 {
			mafiaUpdateUserRole(s, m, db, username, "neutral", counts.Neutral)
			counts.Neutral--
			continue
		}
		if counts.Bad > 0 {
			mafiaUpdateUserRole(s, m, db, username, "bad", counts.Bad)
			counts.Bad--
			continue
		}
		if counts.Good > 0 {
			mafiaUpdateUserRole(s, m, db, username, "good", counts.Good)
			counts.Good--
			continue
		}

	}
}

func mafiaUpdateUserRole(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB, username string, townieType string, count int) {
	switch townieType {
	case "neutral":
		role := selectRandomRole(s, m, db, neutralRoles, username)
		yourRole, exists := constructPlayerRole(role)
		if !exists {
			fmt.Print("There isn't a role associated with that role, ", role)
		}
		yourRole.classMessage(s, m, username)

	case "good":
		role := selectRandomRole(s, m, db, goodRoles, username)
		yourRole, exists := constructPlayerRole(role)
		if !exists {
			fmt.Print("There isn't a role associated with that role, ", role)
		}
		yourRole.classMessage(s, m, username)

	case "bad":
		role := selectRandomRole(s, m, db, badRoles, username)
		yourRole, exists := constructPlayerRole(role)
		if !exists {
			fmt.Print("There isn't a role associated with that role, ", role)
		}
		yourRole.classMessage(s, m, username)
	}
	fmt.Print("assigning " + townieType + "character to " + username + "\n")
}

func getmafiaGameLobbyRoles(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT role FROM mafiaPLayers INNER JOIN mafiaGames on mafiaPlayers.GameID = mafiaGames.GameID WHERE mafiaGames.status = 'Lobby'")
	if err != nil {
		return rows, err
	}
	return rows, err
}

func roleNotTaken(db *sql.DB, role string) bool {
	rows, err := getmafiaGameLobbyRoles(db)
	if err != nil {
		fmt.Println("There was an issue getting the game lobby:", err)
	}

	for rows.Next() {
		var usersCol string
		rows.Scan(&usersCol)
		if usersCol == role {
			return false
		}

	}
	return true
}

func selectRandomRole(s *discordgo.Session, m *discordgo.MessageCreate, db *sql.DB, roleArray []string, username string) string {
	ExistingLobby, lobbyerr := ExistingGameWithStatus(db, m.GuildID, "Lobby")
	if lobbyerr != nil {
		fmt.Println("There was an error finding a lobby: ", lobbyerr)
	}
	role := rand.Intn(len(roleArray))
	for !roleNotTaken(db, roleArray[role]) {
		rand.Intn(len(roleArray))
	}
	_, err := db.Exec("UPDATE mafiaPlayers SET role = ? WHERE gameID = ? AND user = ?", roleArray[role], ExistingLobby, username)
	if err != nil {
		fmt.Print("There was an issue updating the role:", err)
	}
	return roleArray[role]
}

func createGameChannels(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if checkIfExistingGameChannels(s, m) {
		return nil
	}

	category, err := s.GuildChannelCreateComplex(m.GuildID, discordgo.GuildChannelCreateData{
		Name:     "Game Channels",
		Type:     discordgo.ChannelTypeGuildCategory,
		ParentID: "", // No parent category (top-level category)
	})

	if err != nil {
		fmt.Println("Error creating game folder:", err)
		return err
	}

	//townChat, err := s.GuildChannelCreate(m.GuildID, townChannel, discordgo.ChannelTypeGuildText)
	townChat, err := s.GuildChannelCreateComplex(m.GuildID, discordgo.GuildChannelCreateData{
		Name:     townChannel,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID, // Set parent category ID
	})
	if err != nil {
		fmt.Println("there was an error creating the townchat: ", err)
		return err
	}
	fmt.Println("Created townchat: " + townChat.Name)

	deadChat, err := s.GuildChannelCreateComplex(m.GuildID, discordgo.GuildChannelCreateData{
		Name:     deadChannel,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID, // Set parent category ID
	})
	if err != nil {
		fmt.Println("there was an error creating the townchat: ", err)
		return err
	}
	fmt.Println("Created townchat: " + deadChat.Name)

	mafiaChat, err := s.GuildChannelCreateComplex(m.GuildID, discordgo.GuildChannelCreateData{
		Name:     mafiaChannel,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID, // Set parent category ID
	})
	if err != nil {
		fmt.Println("there was an error creating the townchat: ", err)
		return err
	}
	fmt.Println("Created townchat: " + mafiaChat.Name)
	return nil
}

func checkIfExistingGameChannels(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	guildChannels, err := s.GuildChannels(m.GuildID)
	if err != nil {
		fmt.Println("There was an error looking up game channels: ", err)
	}
	var count int
	for _, guild := range guildChannels {
		if guild.Name == townChannel || guild.Name == deadChannel || guild.Name == mafiaChannel {
			count++
		}
	}

	return count == 3

}
