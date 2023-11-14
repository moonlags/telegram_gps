package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"telegram/bot/handlers"
	"telegram/bot/types"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/go-github/v56/github"
	"github.com/joho/godotenv"
	"github.com/levigross/grequests"
	log "github.com/sirupsen/logrus"
)

var token string

var dsp *echotron.Dispatcher

func NewBot(chatID int64) echotron.Bot {
	bot := &handlers.Bot{
		ChatID: chatID,
		API:    echotron.NewAPI(token),
	}
	bot.State = bot.HandleMessage

	handlers.Mutex.Lock()
	handlers.Users[chatID] = bot
	handlers.Mutex.Unlock()

	go bot.SelfDestruct(time.After(24*time.Hour), dsp)

	return bot
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.WithError(err).Fatal("failed to load env variables")
	}

	token = os.Getenv("TELEGRAM_TOKEN")
	types.InitGithubConf()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/callback/github", GithubCallBackHandler)

	go func() {
		dsp = echotron.NewDispatcher(token, NewBot)

		if err := dsp.Poll(); err != nil {
			log.WithError(err).Fatal("failed to start a telegram dispathcer")
		}
	}()

	if err := http.ListenAndServe(":55081", r); err != nil {
		log.WithError(err).Fatal("failed to start http server")
	}
}

func GithubCallBackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	chatID, err := strconv.ParseInt(state, 10, 64)
	if err != nil {
		log.WithError(err).WithField("state", state).Error("failed to convert state to int")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := types.GithubConf().Exchange(context.Background(), code)
	if err != nil {
		log.WithError(err).WithField("code", code).Error("failed to exchange")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	client := github.NewClient(types.GithubConf().Client(context.Background(), token))

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		log.WithError(err).Error("failed to get user")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp, err := grequests.Post("http://localhost:50731/api/user/login", &grequests.RequestOptions{
		JSON: map[string]interface{}{"ID": *user.ID},
	})
	if err != nil {
		log.WithError(err).Error("failed to send a request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Close()

	userData := new(types.User)
	if err := resp.JSON(userData); err != nil {
		log.WithError(err).Error("failed to parse json response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if userData.GithubID != *user.ID {
		log.WithField("userdata id", userData.GithubID).WithField("github id", *user.ID).Error("ids dont match")
		if _, err := w.Write([]byte("Something went wrong try again later!")); err != nil {
			log.WithError(err).Error("failed to write a response")
		}
		return
	}

	handlers.Mutex.Lock()

	handlers.Users[chatID].DeviceIMEI = userData.IMEI
	handlers.Users[chatID].User = user
	handlers.Users[chatID].LoggedIn = true

	bot := handlers.Users[chatID]
	if bot.DeviceIMEI == "" {
		if _, err := bot.SendMessage("Hello, its seems that you have not linked any device to your profile! Please enter your devices IMEI: ", bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}

		bot.State = bot.HandleImeiInput
	} else {
		if _, err := bot.SendMessage("Welcome back!\n /position - get last device postition\n /sleep - set device in sleep mode", bot.ChatID, nil); err != nil {
			log.WithError(err).Error("failed to send a message")
		}

		bot.State = bot.HandleLoggedIn
	}

	handlers.Mutex.Unlock()

	if _, err := w.Write([]byte("Now you are logged in, you can return to telegram!")); err != nil {
		log.WithError(err).Error("failed to write a response")
	}
}
