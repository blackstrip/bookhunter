package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// channelInfo will return the last available book id, channel id and access hash.
func channelInfo(ctx context.Context, client *telegram.Client, config *Config) (*tgChannelInfo, error) {
	var channelID int64
	var accessHash int64
	var err error

	if id, e := strconv.ParseInt(config.ChannelID, 10, 64); e == nil {
		channelID = id
		accessHash, err = privateChannelInfo(ctx, client, id)
	} else {
		channelID, accessHash, err = publicChannelInfo(ctx, client, config.ChannelID)
	}

	if err != nil {
		return nil, err
	}

	last, err := queryLatestMsgID(ctx, client, channelID, accessHash)
	if err != nil {
		return nil, err
	}

	return &tgChannelInfo{
		id:         channelID,
		accessHash: accessHash,
		lastMsgID:  last,
	}, nil
}

// Query access hash for private channel.
func privateChannelInfo(ctx context.Context, client *telegram.Client, channelID int64) (accessHash int64, err error) {
	c, err := client.API().ContactsGetContacts(ctx, channelID)
	if err != nil {
		return
	}

	if contacts, ok := c.(*tg.ContactsContacts); ok {
		for _, u := range contacts.Users {
			if user, ok := u.(*tg.User); ok {
				accessHash = user.AccessHash
				return
			}
		}
	}

	err = errors.New("couldn't find access hash")
	return
}

// Query public channel by its name.
func publicChannelInfo(ctx context.Context, client *telegram.Client, channelName string) (channelID, accessHash int64, err error) {
	username, err := client.API().ContactsResolveUsername(ctx, channelName)
	if err != nil {
		return
	}

	if len(username.Chats) == 0 {
		err = fmt.Errorf("you are not belong to channel: %s", channelName)
		return
	}

	for _, chat := range username.Chats {
		// Try to find the related channel.
		if channel, ok := chat.(*tg.Channel); ok {
			channelID = channel.ID
			accessHash = channel.AccessHash
			return
		}
	}

	err = fmt.Errorf("couldn't find channel id and hash for channel: %s", channelName)
	return
}

// queryLatestMsgID from the given channel info.
func queryLatestMsgID(ctx context.Context, client *telegram.Client, channelID, accessHash int64) (int64, error) {
	request := &tg.MessagesSearchRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		Filter:   &tg.InputMessagesFilterEmpty{},
		Q:        "",
		OffsetID: -1,
		Limit:    1,
	}

	last := -1
	search, err := client.API().MessagesSearch(ctx, request)
	if err != nil {
		return 0, err
	}

	channelInfo, ok := search.(*tg.MessagesChannelMessages)
	if !ok {
		return 0, err
	}

	for _, msg := range channelInfo.Messages {
		if msg != nil {
			last = msg.GetID()
			break
		}
	}

	if last <= 0 {
		return 0, errors.New("couldn't find last message id")
	}

	return int64(last), nil
}