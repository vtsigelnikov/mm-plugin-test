package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	botUserName    = "remindbot"
	botDisplayName = "Remindbot"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	router    *mux.Router
	botUserId string
	running   bool
	emptyTime time.Time

	ServerConfig *model.Config

	readFile func(path string) ([]byte, error)
	locales  map[string]string
}

func NewPlugin() *Plugin {
	return &Plugin{
		readFile: os.ReadFile,
		locales:  make(map[string]string),
	}
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	p.ServerConfig = p.API.GetConfig()
	p.router = p.InitAPI()
	p.emptyTime = time.Time{}.AddDate(1, 1, 1)

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: "Created by the GitHub plugin.",
	}, pluginapi.ProfileImagePath("assets/icon.png"))
	if err != nil {
		return fmt.Errorf("%s: %w", "failed to ensure remind bot", err)

	}
	p.botUserId = botID

	err = p.registerCommand()
	if err != nil {
		return fmt.Errorf("%s: %w", "failed to register command", err)
	}

	if err := p.TranslationsPreInit(); err != nil {
		return fmt.Errorf("%s: %w", "failed to initialize translations", err)
	}
	p.Run()

	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.Stop()
	return nil
}
