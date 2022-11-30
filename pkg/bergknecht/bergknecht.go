package bergknecht

import (
	"errors"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/config"
	"github.com/Nerdbergev/Bergknecht/pkg/eventhandler"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var handler []eventhandler.BergEventHandler

func init() {
	//handler = append(handler, )
}

func doLogin(conf config.Config) (*mautrix.Client, error) {
	client, err := mautrix.NewClient(conf.Homserver, "", "")
	if err != nil {
		return nil, errors.New("Error creating Client: " + err.Error())
	}

	var ident mautrix.UserIdentifier
	ident.User = conf.Username
	ident.Type = "m.id.user"

	var reqLog mautrix.ReqLogin
	reqLog.Identifier = ident
	reqLog.Password = conf.Password
	reqLog.Type = "m.login.password"
	reqLog.StoreCredentials = true
	reqLog.StoreHomeserverURL = true

	_, err = client.Login(&reqLog)
	if err != nil {
		return nil, errors.New("Error logging in: " + err.Error())
	}

	return client, nil
}

func joinRooms(client *mautrix.Client, conf config.Config) error {
	for _, r := range conf.Rooms {
		_, err := client.JoinRoom(r, conf.Homserver, nil)
		if err != nil {
			return errors.New("Error joing Room " + r + ": " + err.Error())
		}
	}
	return nil
}

func isinRoomList(roomID string, roomList []string) bool {
	for _, r := range roomList {
		if strings.Compare(r, roomID) == 0 {
			return true
		}
	}
	return false
}

func RunBot(conf config.Config) error {
	client, err := doLogin(conf)
	if err != nil {
		return errors.New("Error logging in: " + err.Error())
	}
	err = joinRooms(client, conf)
	if err != nil {
		return errors.New("Error joining in: " + err.Error())
	}

	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEvent(func(source mautrix.EventSource, evt *event.Event) {
		if isinRoomList(evt.RoomID.String(), conf.Rooms) {
			for _, h := range handler {
				handled := h(client, source, evt)
				if handled {
					break
				}
			}
		}
	})
	err = client.Sync()
	if err != nil {
		return errors.New("Error syncing: " + err.Error())
	}

	return nil
}
