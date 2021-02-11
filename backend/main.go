package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/lafin/clubhouseapi"
)

func login() {
	_ = godotenv.Load()
	response, err := clubhouseapi.StartPhoneNumberAuth(os.Getenv("PHONE_NUMBER"))
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	if !response.Success {
		return
	}
}

func auth(verificationCode string) {
	_ = godotenv.Load()
	response, err := clubhouseapi.CompletePhoneNumberAuth(os.Getenv("PHONE_NUMBER"), verificationCode)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	if !response.Success {
		return
	}
	env, _ := godotenv.Read()
	env["ACCESS_TOKEN"] = response.AccessToken
	env["REFRESH_TOKEN"] = response.RefreshToken
	env["USER_ID"] = strconv.Itoa(response.UserProfile.UserID)
	_ = godotenv.Write(env, ".env")
}

func channels() (clubhouseapi.GetChannelsResponse, error) {
	var credentials = map[string]string{
		"CH-UserID":     os.Getenv("USER_ID"),
		"Authorization": fmt.Sprintf(`Bearer %s`, os.Getenv("ACCESS_TOKEN")),
	}
	clubhouseapi.AddCredentials(credentials)
	return clubhouseapi.GetChannels()
}

func refresh() {
	response, err := clubhouseapi.RefreshToken(os.Getenv("REFRESH_TOKEN"))
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	env, _ := godotenv.Read()
	env["ACCESS_TOKEN"] = response.Access
	env["REFRESH_TOKEN"] = response.Refresh
	_ = godotenv.Write(env, ".env")
}

func userIsAlreadyInChannel(channel clubhouseapi.Channel, userID int) bool {
	for _, user := range channel.Users {
		if user.UserID == userID {
			return true
		}
	}
	return false
}

func autoRefresh() {
	ticker := time.NewTicker(60 * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			refresh()
			fmt.Print("@")
			_ = godotenv.Overload()
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func online() {
	_ = godotenv.Load()
	refresh()
	for {
		time.Sleep(60 * time.Second)
		refresh()
		fmt.Print("@")
	}
}

func joinEveryRoom() {
	_ = godotenv.Load()
	refresh()
	go autoRefresh()

	for {
		response, err := channels()
		if err != nil {
			log.Fatalln(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}
		sort.Slice(response.Channels, func(i, j int) bool { return response.Channels[i].NumAll < response.Channels[j].NumAll })
		for _, channel := range response.Channels {
			fmt.Println(channel.ChannelID, channel.Channel, channel.Topic, channel.Club.Name, channel.NumAll, channel.NumSpeakers)
		}
		for _, channel := range response.Channels {
			userID, _ := strconv.ParseInt(os.Getenv("USER_ID"), 10, 32)
			if !userIsAlreadyInChannel(channel, int(userID)) {
				_, err := clubhouseapi.JoinChannel(channel.Channel)
				if err != nil {
					log.Fatalln(err.Error())
					break
				}
				fmt.Print("+")
				time.Sleep(4 * time.Second)
			} else {
				_, err := clubhouseapi.ActivePing(channel.Channel)
				if err != nil {
					log.Fatalln(err.Error())
					break
				}
				fmt.Print(".")
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func main() {
	actionPtr := flag.String("action", "", "example: login, auth, online, join-every-room")
	verificationCodePtr := flag.String("verificationCode", "", "verification code")
	flag.Parse()

	switch *actionPtr {
	case "login":
		login()
	case "auth":
		auth(*verificationCodePtr)
	case "online":
		online()
	case "join-every-room":
		joinEveryRoom()
	}
}
