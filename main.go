package main

import (
	//"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	dat, err := ioutil.ReadFile("token")
	token := string(dat)
	fmt.Print(token)
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	dg.ChannelMessageSend("387582363590459412", "YUI is online now.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "苟利国家生死以" {
		s.ChannelMessageSend(m.ChannelID, "岂因祸福避趋之")
	}
}
