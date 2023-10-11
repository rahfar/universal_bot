package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jessevdk/go-flags"

	"github.com/rahfar/familybot/src/apiclient"
	"github.com/rahfar/familybot/src/bot"
)

var opts struct {
	Telegram struct {
		Token            string `long:"token" env:"TOKEN" description:"telegram bot token" default:"test"`
		GroupID          int64  `long:"group" env:"GROUP" description:"group id"`
		AllowedUsernames string `long:"allowedusernames" env:"ALLOWEDUSERNAMES" description:"list of usernames that will have access to the bot" default:""`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	WeatherAPI struct {
		Key    string `long:"key" env:"KEY"`
		Cities string `long:"cities" env:"CITIES"`
	} `group:"weatherapi" namespace:"weatherapi" env-namespace:"WEATHERAPI"`
	CurrencyAPI struct {
		Key string `long:"key" env:"KEY"`
	} `group:"currencyapi" namespace:"currencyapi" env-namespace:"CURRENCYAPI"`
	OpenaiAPI struct {
		Key      string `long:"key" env:"KEY"`
		GPTModel string `long:"gptmodel" env:"GPTMODEL" default:"gpt-3.5-turbo"`
	} `group:"openaiapi" namespace:"openaiapi" env-namespace:"OPENAIAPI"`
	Dbg bool `long:"debug" env:"DEBUG" description:"debug mode"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	slog.SetDefault(logger)
	tgbotapi.SetLogger(log.Default())

	if _, err := flags.Parse(&opts); err != nil {
		slog.Error("Error parsing options")
		panic(err)
	}

	bot_api, err := tgbotapi.NewBotAPI(opts.Telegram.Token)
	if err != nil {
		slog.Error("could not initialize bot", "err", err)
		panic(err)
	}

	bot_api.Debug = opts.Dbg

	slog.Info("bot is authorized", "bot-username", bot_api.Self.UserName)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	openaiHttpClient := &http.Client{Timeout: 60 * time.Second}

	anekdotAPI := &apiclient.AnecdoteAPI{HttpClient: httpClient}
	exchangeAPI := &apiclient.ExchangeAPI{ApiKey: opts.CurrencyAPI.Key, HttpClient: httpClient}
	kommerstantAPI := &apiclient.KommersantAPI{HttpClient: httpClient}
	openaiAPI := &apiclient.OpenaiAPI{ApiKey: opts.OpenaiAPI.Key, HttpClient: openaiHttpClient, GPTModel: opts.OpenaiAPI.GPTModel}
	weatherAPI := &apiclient.WeatherAPI{ApiKey: opts.WeatherAPI.Key, Cities: opts.WeatherAPI.Cities, HttpClient: httpClient}

	b := bot.Bot{
		Token:            opts.Telegram.Token,
		Dbg:              opts.Dbg,
		AllowedUsernames: strings.Split(opts.Telegram.AllowedUsernames, ","),
		GroupID:          opts.Telegram.GroupID,
		Commands:         bot.Commands,
		AskGPTCache:      expirable.NewLRU[string, []apiclient.GPTResponse](1000, nil, time.Minute*30),
		AnekdotAPI:       anekdotAPI,
		ExchangeAPI:      exchangeAPI,
		KommersantAPI:    kommerstantAPI,
		OpenaiAPI:        openaiAPI,
		WeatherAPI:       weatherAPI,
		TGBotAPI:         bot_api,
	}
	b.Run()
}
