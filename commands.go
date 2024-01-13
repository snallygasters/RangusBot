package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func handleLethalCompany(s *discordgo.Session, m *discordgo.MessageCreate) {

	_, err := s.State.Channel((m.ChannelID))
	if err != nil {
		return
	}

	for i := 0; i < 5; i++ {
		s.ChannelMessageSend(m.ChannelID, "@everyone https://tenor.com/miz2U0CJdmt.gif")
	}
}

func handleBegoneCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.State.Channel((m.ChannelID))
	if err != nil {
		return
	}

	filePath := getSoundFilePath("begone")

	parts := strings.Fields(m.Content)

	if len(parts) == 2 {
		username := parts[1]
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("BEGONE @%s", username))
		fmt.Println("Kicking: ", username)

		memberToKick, err := searchUserByUsername(s, m.GuildID, username)
		if err != nil {
			fmt.Printf("%s is not a user", username)
			return
		}
		playAudioFile(s, m.GuildID, m.ChannelID, filePath, memberToKick)
	}
}
