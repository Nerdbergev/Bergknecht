package bergknecht

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/Nerdbergev/Bergknecht/pkg/berghandler"
	"github.com/Nerdbergev/Bergknecht/pkg/config"
	"github.com/Nerdbergev/Bergknecht/pkg/handlers/bestellungHandler"
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var handlers []berghandler.BergEventHandler
var startup time.Time

func init() {
	h := bestellungHandler.BestellungHandler{}
	handlers = append(handlers, &h)
	startup = time.Now()
}

func doLogin(conf config.Config) (*mautrix.Client, error) {
	client, err := mautrix.NewClient(conf.Serversettings.Homserver, "", "")
	if err != nil {
		return nil, errors.New("Error creating Client: " + err.Error())
	}

	var ident mautrix.UserIdentifier
	ident.User = conf.Serversettings.Username
	ident.Type = "m.id.user"

	var reqLog mautrix.ReqLogin
	reqLog.Identifier = ident
	reqLog.Password = conf.Serversettings.Password
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
	for _, r := range conf.Serversettings.Rooms {
		_, err := client.JoinRoom(r, conf.Serversettings.Homserver, nil)
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
	logger := zap.Must(conf.LoggerSettings.Build())
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infow("Logging in")
	client, err := doLogin(conf)
	if err != nil {
		return errors.New("Error logging in: " + err.Error())
	}
	sugar.Infow("Joining Rooms")
	err = joinRooms(client, conf)
	if err != nil {
		return errors.New("Error joining in: " + err.Error())
	}

	sugar.Infow("Setting up Storage")
	sm := storage.CreateStorageManager(conf.StorageSettings)
	defer sm.DeleteCache()

	rand.Seed(time.Now().UnixNano())

	he := berghandler.HandlerEssentials{Client: client, Logger: sugar, Storage: sm}

	sugar.Infow("Loading Handler Data")
	for _, h := range handlers {
		err := h.LoadData(he)
		if err != nil {
			sugar.Errorw("Hanlder unable to load data", "handlername", h.GetName(), "error", err)
		}
	}

	sugar.Infow("Starting Syncer")
	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEvent(func(source mautrix.EventSource, evt *event.Event) {
		if (evt.Sender != client.UserID) && (isinRoomList(evt.RoomID.String(), conf.Serversettings.Rooms) && (evt.Timestamp >= startup.UnixMilli())) {
			for _, h := range handlers {
				handled := h.Handle(he, source, evt)
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
