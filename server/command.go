package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const CommandTrigger = "remind"

func (p *Plugin) registerCommand() error {
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          CommandTrigger,
		AutoComplete:     true,
		AutoCompleteHint: "[@someone or ~channel] [what] [when]",
		AutoCompleteDesc: "Set a reminder",
	}); err != nil {
		return fmt.Errorf("%s: %w", "failed to register command", err)
	}

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	user, uErr := p.API.GetUser(args.UserId)
	if uErr != nil {
		return &model.CommandResponse{}, uErr
	}

	T, locale := p.translation(user)
	location := p.location(user)
	command := strings.Trim(args.Command, " ")

	if strings.Trim(command, " ") == "/"+CommandTrigger {
		p.InteractiveSchedule(args.TriggerId, user)
		return &model.CommandResponse{}, nil
	}

	if strings.HasSuffix(command, T("help")) {
		post := model.Post{
			ChannelId: args.ChannelId,
			UserId:    p.botUserId,
			Message:   T("help.response"),
		}
		p.API.SendEphemeralPost(user.Id, &post)
		return &model.CommandResponse{}, nil
	}

	if strings.HasSuffix(command, T("list")) {
		p.API.SendEphemeralPost(user.Id, p.ListReminders(user, args.ChannelId))
		return &model.CommandResponse{}, nil
	}

	// clear all reminders for current user
	if strings.HasSuffix(command, "__clear") {
		post := model.Post{
			ChannelId: args.ChannelId,
			UserId:    p.botUserId,
			Message:   p.DeleteReminders(user),
		}
		p.API.SendEphemeralPost(user.Id, &post)
		return &model.CommandResponse{}, nil
	}

	// display the plugin version
	if strings.HasSuffix(command, "__version") {
		post := model.Post{
			ChannelId: args.ChannelId,
			UserId:    p.botUserId,
			Message:   manifest.Version,
		}
		p.API.SendEphemeralPost(user.Id, &post)
		return &model.CommandResponse{}, nil
	}

	// display the locale & location of user
	if strings.HasSuffix(command, "__user") {
		post := model.Post{
			ChannelId: args.ChannelId,
			UserId:    p.botUserId,
			Message:   "locale: " + locale + "\nlocation: " + location.String(),
		}
		p.API.SendEphemeralPost(user.Id, &post)
		return &model.CommandResponse{}, nil
	}

	payload := strings.Trim(strings.Replace(command, "/"+CommandTrigger, "", -1), " ")
	request := ReminderRequest{
		TeamId:   args.TeamId,
		Username: user.Username,
		Payload:  payload,
		Reminder: Reminder{},
	}
	reminder, err := p.ScheduleReminder(&request, args.ChannelId, user.Id)

	if err != nil {
		post := model.Post{
			ChannelId: args.ChannelId,
			UserId:    p.botUserId,
			Message:   T("exception.response"),
		}
		p.API.SendEphemeralPost(user.Id, &post)
		return &model.CommandResponse{}, nil
	}

	p.API.SendEphemeralPost(user.Id, reminder)
	return &model.CommandResponse{}, nil

}
