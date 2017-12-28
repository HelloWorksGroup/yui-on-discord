package main

import (
	"fmt"
	//"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"

	"github.com/anthonynsimon/bild/effect"
	//"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
)

const version string = "YUI-Zero Alpha0004"

var debug_channel string
var talking_channel string
var special_channel1 string

var DS *discordgo.Session

func main() {
	rand.Seed(time.Now().UnixNano())
	viper.SetDefault("token", 0)
	viper.SetDefault("debugChannel", 0)
	viper.SetDefault("talkingChannel", 0)
	viper.SetDefault("specialChannel", 0)
	viper.SetDefault("oldversion", "unknown")
	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	token := viper.Get("token").(string)
	fmt.Println("token=" + token)

	debug_channel = viper.Get("debugChannel").(string)
	fmt.Println("debugChannel=" + debug_channel)

	talking_channel = viper.Get("talkingChannel").(string)
	fmt.Println("talkingChannel=" + talking_channel)

	special_channel1 = viper.Get("specialChannel").(string)
	fmt.Println("specialChannel=" + special_channel1)

	oldVersion := viper.Get("oldversion").(string)
	var newVersion bool
	if oldVersion == version {
		newVersion = false
	} else {
		newVersion = true
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	DS = dg
	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)
	dg.AddHandler(typingStart)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	if newVersion {
		dg.ChannelMessageSend(debug_channel, "YUI升级了呢！\n旧的版本是:"+oldVersion+"\n现在升级到了:"+version)
		viper.Set("oldversion", version)
		viper.WriteConfig()
	}
	viper.Reset()
	fmt.Println("YUI desu.")

	msg, _ := dg.ChannelMessageSend(debug_channel, "YUI is online now.")
	go func() {
		<-time.After(time.Second * 5)
		dg.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	// s.UpdateStatus(0, "Artifact Idiot")
	s.UpdateStatus(0, "Sword Art Online")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	channel, _ := s.Channel(m.ChannelID)
	channelName := channel.Name
	fmt.Printf("<" + m.ChannelID + ">[" + channelName + "]" + m.Author.Username + ":" + m.Content + "\n")
	if m.ChannelID == special_channel1 {
		for _, v := range m.Attachments {
			if v.Width > 0 {
				fmt.Printf("图片尺寸:%dx%d\n", v.Width, v.Height)
				fmt.Println(v)
				go func() {
					res, err := http.Get(v.URL)

					if err != nil {
						panic(err)
					}
					f, err := os.Create(v.Filename)
					if err != nil {
						panic(err)
					}
					io.Copy(f, res.Body)
					f.Close()
					res.Body.Close()

					img, err := imgio.Open(v.Filename)
					if err != nil {
						panic(err)
					}

					k := 300 / math.Max(float64(img.Bounds().Max.X), float64(img.Bounds().Max.Y))
					w, h := int(math.Ceil(k*float64(img.Bounds().Max.X))), int(math.Ceil(k*float64(img.Bounds().Max.Y)))
					result := effect.Median(transform.Resize(transform.Resize(img, w/8, h/8, transform.NearestNeighbor), w, h, transform.NearestNeighbor), 2)
					if err := imgio.Save("output.jpg", result, imgio.JPEGEncoder(15)); err != nil {
						panic(err)
					}
					os.Remove(v.Filename)
					file, _ := os.OpenFile("output.jpg", os.O_RDONLY, 0)
					dcuser, _ := s.User(m.Author.ID)
					s.ChannelFileSend(talking_channel, "女装.jpg", file)
					s.ChannelMessageSend(talking_channel, dcuser.Mention()+"在女装频道发布了图片，缩略图打码如上")
				}()
			}
		}
	}

	// 愿YUI寿与天齐
	if m.Content == "苟利国家生死以" {
		s.ChannelMessageSend(m.ChannelID, "岂因祸福避趋之")
	}
	if m.ChannelID == debug_channel {
		if strings.Contains(m.Content, "yui 吃鸡") {
			if GameState == "idle" {
				GameNewRoom()
				talkto(m.ChannelID, "YUI已建立好了可供大家游玩的房间哦", 10)
			} else {
				talkto(m.ChannelID, "上一个战局还没有结束，请大家稍稍等候一下啦", 10)
			}
		}
		if strings.Contains(m.Content, "yui 关闭战局") {
			if GameState == "idle" {
				talkto(m.ChannelID, "没有正在进行的战局哦，现在可以开始新的游戏呢", 10)
			} else {
				GameClear()
				talkto(m.ChannelID, "好的，YUI已经帮你把战局关闭了哟", 10)
			}
		}
	}
	if GameState != "idle" && m.ChannelID == GameChannel.ID {
		GameRoomMessageHandler(s, m)
	}
	if GameState != "idle" && channel.Type == discordgo.ChannelTypeDM && IsPlayer(m.Author.ID) {
		GamePrivateMessageHandler(s, m)
	}
}

func typingStart(s *discordgo.Session, m *discordgo.TypingStart) {
	if m.UserID == s.State.User.ID {
		return
	}
	if m.ChannelID == debug_channel {
		user, _ := s.User(m.UserID)
		msg, _ := s.ChannelMessageSend(m.ChannelID, "侦测到在途聚变打击\n来源:"+user.Mention()+"\n发射时间:"+time.Unix(int64(m.Timestamp), 0).Format("1月2日15时04分05秒"))
		go func() {
			<-time.After(time.Second * 7)
			s.ChannelMessageDelete(msg.ChannelID, msg.ID)
		}()
	}
}

func talkto(channel string, str string, speed int) {
	go func(channel string, str string, speed int) {
		<-time.After(time.Millisecond * time.Duration(speed) * time.Duration(len(str)))
		DS.ChannelMessageSend(channel, str)
	}(channel, str, speed)
}
