// game.go
// 吃鸡相关逻辑和接口
package main

import (
	"bytes"
	"strings"
	"time"
	//"math/rand"
	"fmt"

	"github.com/Guitarbum722/align"
	"github.com/HWSkynet/cpgame"
	"github.com/bwmarrin/discordgo"
)

type Players []*cpgame.Player

var GameState string = "idle"
var GameChannel *discordgo.Channel

var id2dmChannel map[string]string

var playerList *cpgame.PlayerList

func reply(s *discordgo.Session, m *discordgo.MessageCreate, words string) {
	s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+words)
}

func gameNotice(s *discordgo.Session, words string) {
	s.ChannelMessageSend(GameChannel.ID, words)
}

func GameNewRoom() {
	var err error
	GameChannel, err = DS.GuildChannelCreate("377366788322623491", "GAMEROOM-eat-chicken", "text")
	if err != nil {
		panic(err)
	}
	GameState = "ready"
	playerList = cpgame.NewPlayerList()
	DS.ChannelMessageSend(GameChannel.ID, "```\n命令列表:\njoin:加入战局\nexit:离开战局\n\n已经加入战局的玩家之后的交互都需通过PM YUI完成\nready:准备游戏或取消准备\n```")
	id2dmChannel = make(map[string]string, 100)
}

func GameClear() {
	DS.ChannelDelete(GameChannel.ID)
	GameState = "idle"
}

func GameStart() {
	if len(*playerList) > 0 {
		if playerList.CountReady() < len(*playerList) {
			gameNotice(DS, fmt.Sprintf("已准备人数：%d/%d，还有%d名玩家没有准备", playerList.CountReady(), len(*playerList), len(*playerList)-playerList.CountReady()))
			return
		}
	} else {
		DS.ChannelMessageSend(GameChannel.ID, "战局里面没有玩家，是不能开始游戏的哦")
		return
	}
	DS.ChannelMessageSend(GameChannel.ID, ":white_check_mark: 游戏已经开始:white_check_mark: "+
		"\n:large_orange_diamond: 本频道转为游戏全局信息发布频道"+
		"\n:large_blue_circle: 可通过**status**命令查询当前游戏进程(10秒内最多查询一次)"+
		"\n\n第一回合将在10秒后开始，请大家切换到"+DS.State.User.Mention()+"的私人频道做好准备。")
	GameState = "gaming"
	cpgame.GameClockStart()
}

var cmdTimer int = 0

func GameRoomMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if GameState == "ready" {
		switch m.Content {
		case "join":
			if !playerList.IsExist(m.Author.ID) {
				playerList.Add(m.Author.ID)
				playerList.ID(m.Author.ID).Name = m.Author.String()

				reply(s, m, fmt.Sprintf("加入战局，当前大厅人数：%d 已准备人数：%d/%d", len(*playerList), playerList.CountReady(), len(*playerList)))
			} else {
				reply(s, m, "你已经加入了战局了哦，请耐心等待游戏组织者开始游戏")
			}
		case "exit":
			if !playerList.IsExist(m.Author.ID) {
				reply(s, m, "你已经没有在战局中了哦")
			} else {
				playerList.Remove(m.Author.ID)
				reply(s, m, fmt.Sprintf("退出战局，当前大厅人数：%d 已准备人数：%d/%d", len(*playerList), playerList.CountReady(), len(*playerList)))
			}

		case "link start":
			GameStart()
		}
	}
	if GameState == "gaming" {
		switch m.Content {
		case "status":
			if cmdTimer > 0 {
				break
			}
			go func() {
				cmdTimer = 5
				<-time.After(time.Second * 5)
				cmdTimer = 0
			}()
			information := fmt.Sprintf("游戏已进行%d分钟,存活玩家数%d/%d", cpgame.GameMinutes, playerList.CountLive(), len(*playerList)) +
				"\n`Name`,`Kill`,`Assist`\n"
			for _, v := range *playerList {
				if v.Life > 0 {
					information += ":dress: "
				} else {
					information += ":skull: "
				}
				information += "`" + v.Name + "`" + fmt.Sprintf(",`%d`,`%d`\n", v.Killed, v.Assist)
			}
			reply(s, m, gameStatuNice(information))

		}
	}
}

func IsPlayer(id string) bool {
	return playerList.IsExist(id)
}

func GamePrivateMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if GameState == "ready" {
		switch m.Content {
		case "ready":
			// 将用户添加到id映射到pm频道的map中
			// 因为pm频道id没有公开的api可以获取
			// 用户退出后也可以不删除，因为消息前面已经过滤掉了非玩家的信息
			if _, ok := id2dmChannel[m.Author.ID]; ok {
				// 用户已经在map中了 do nothing
			} else {
				id2dmChannel[m.Author.ID] = m.ChannelID
				reply(s, m, "欢迎使用YUI智能终端。\n"+
					"由于这个世界限制了YUI的机能，YUI每分钟只能发送100条消息，所以请不要连续发送过多指令。\n"+
					"YUI在处理完你发送的上一条指令之前，将不会接收新的指令。YUI对此非常抱歉，希望你能理解。")
			}
			playerList.ImReady(m.Author.ID)
			if playerList.ID(m.Author.ID).State != "ready" {
				reply(s, m, "取消了准备")
				gameNotice(s, m.Author.Mention()+fmt.Sprintf("取消了准备呢，已准备人数：%d/%d", playerList.CountReady(), len(*playerList)))
			} else {
				reply(s, m, "准备好了呢")
				gameNotice(s, m.Author.Mention()+fmt.Sprintf("已经准备好了，已准备人数：%d/%d", playerList.CountReady(), len(*playerList)))
			}
		}
	}
	if GameState == "gaming" {
		info := playerList.ID(m.Author.ID).InputParse(m.Content)
		fmt.Println(info)
		if len(info.Text) > 0 {
			gameNotice(s, m.Author.Mention()+info.Text)
		}
	}
}

func gameStatuNice(str string) string {
	input := strings.NewReader(str)
	output := bytes.NewBufferString("")

	aligner := align.NewAlign(input, output, ",", align.TextQualifier{})
	aligner.UpdatePadding(
		align.PaddingOpts{
			Justification: align.JustifyCenter,
			ColumnOverride: map[int]align.Justification{
				1: align.JustifyLeft,
			},
			Pad: 3,
		})
	aligner.Align()
	return output.String()
}
