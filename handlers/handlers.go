package handlers

import (
	"fmt"
	"strings"
	"sync"
	"telegram/bot/types"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/dustin/go-humanize"
	"github.com/google/go-github/v56/github"
	"github.com/levigross/grequests"
	log "github.com/sirupsen/logrus"
)

var (
	Users = make(map[int64]*Bot)
	Mutex = new(sync.Mutex)
)

type Bot struct {
	ChatID     int64
	State      types.StateFn
	DeviceIMEI string
	User       *github.User
	LoggedIn   bool
	echotron.API
}

func (bot *Bot) Update(update *echotron.Update) {
	log.WithField("text", update.Message.Text).Info("new message")
	bot.State = bot.State(update)
}

func (bot *Bot) SelfDestruct(timech <-chan time.Time, dsp *echotron.Dispatcher) {
	<-timech

	dsp.DelSession(bot.ChatID)
}

func (bot *Bot) HandleMessage(update *echotron.Update) types.StateFn {
	if strings.HasPrefix(update.Message.Text, "/login") {
		authUrl := types.GithubConf().AuthCodeURL(fmt.Sprintf("%d", bot.ChatID))

		if _, err := bot.SendMessage(fmt.Sprint("Please follow this link to login and then say \"LOGIN\", when you are done!\n ", authUrl), bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}

		return bot.HandleMessage
	} else {
		if _, err := bot.SendMessage("Login first using /login to use the bot", bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}

		return bot.HandleMessage
	}

}

func (bot *Bot) HandleLoggedIn(update *echotron.Update) types.StateFn {
	if strings.HasPrefix(update.Message.Text, "/help") {
		if _, err := bot.SendMessage("Command list:\n /help - dislpays this list\n /position - get last device postition\n /sleep - set device in sleep mode", bot.ChatID, nil); err != nil {

			log.WithError(err).Error("failed to send a message")
		}
	} else if strings.HasPrefix(update.Message.Text, "/position") {
		resp, err := grequests.Get(fmt.Sprintf("http://localhost:50731/api/user/%d/device/%s", *bot.User.ID, bot.DeviceIMEI), nil)
		if err != nil {
			log.WithError(err).Error("failed to send a request")
			if _, err := bot.SendMessage("Something went wrong. Please try again later!", bot.ChatID, nil); err != nil {
				log.WithError(err).Error("failed to send a message")
			}
			return bot.HandleLoggedIn
		}

		defer resp.Close()

		fmt.Println(resp.String())

		deviceData := new(types.DeviceData)

		if err := resp.JSON(deviceData); err != nil {
			log.WithError(err).Error("failed to get device data")
			return bot.HandleLoggedIn
		}

		if _, err := bot.SendMessage(fmt.Sprintf("Info for Device: %s\n Batter Power - %d\n Is Charging - %v\n Last Packet Time - %s",
			deviceData.IMEI, deviceData.BatteryPower, deviceData.InChargingState, humanize.Time(time.Unix(deviceData.LastStatusPacketTime, 0))),
			bot.ChatID, nil); err != nil {

			log.WithError(err).Error("failed to send a message")
		}

		if len(deviceData.Posititions) > 0 {
			lastPosition := deviceData.Posititions[len(deviceData.Posititions)-1]

			if _, err := bot.SendLocation(bot.ChatID, float64(lastPosition.Latitude), float64(lastPosition.Longitude), nil); err != nil {
				log.WithError(err).Error("failed to send a location")
			}
		}
	}

	return bot.HandleLoggedIn
}

func (bot *Bot) HandleImeiInput(update *echotron.Update) types.StateFn {
	resp, err := grequests.Post("http://localhost:50731/api/user/link", &grequests.RequestOptions{
		JSON: map[string]interface{}{"ID": *bot.User.ID, "IMEI": update.Message.Text},
	})
	if err != nil {
		log.WithError(err).Error("failed to send a request")
		if _, err := bot.SendMessage("Something went wrong. Please check your provided imei and try again!", bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}
		return bot.HandleImeiInput
	}

	defer resp.Close()

	if resp.StatusCode != 200 {
		if _, err := bot.SendMessage("Something went wrong. Please check your provided imei and try again!", bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}
		return bot.HandleImeiInput

	}

	bot.DeviceIMEI = update.Message.Text

	if _, err := bot.SendMessage("Succesefully linked your profile to device!\n Now try /help :)", bot.ChatID, nil); err != nil {
		log.WithError(err).Error("failed to send a message")
	}

	return bot.HandleLoggedIn
}
