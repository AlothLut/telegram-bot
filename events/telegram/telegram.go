package telegram

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strconv"
    "time"
    "user-handler-bot/clients/telegram"
    "user-handler-bot/events"
    "user-handler-bot/helpers"
    "user-handler-bot/storage"
)

const checkAutoAcceptRequestEnableFileStatus = ".auto_accept_status"

type Handler struct {
    client                  *telegram.Client
    storage                 storage.Storage
    offset                  int
    autoAcceptRequestEnable bool
    nextSetSendMsg          string
    processSendingMessage   chan string
    countRequests           int
    lastInlineKeyBoardId    int
    setNewTimeForSentMessageToAll bool
}

type DelayedRequest struct {
    Type string
    Text string
    Meta *telegram.ChatJoinRequest
}

func New(client *telegram.Client, storage storage.Storage) *Handler {
    return &Handler{
        client: client,
        storage: storage,
        autoAcceptRequestEnable: checkAutoAcceptRequestEnable(),
        nextSetSendMsg: "",
        countRequests: 0,
        setNewTimeForSentMessageToAll: false,
    }
}

func checkAutoAcceptRequestEnable() bool {
    _, err := os.Stat(checkAutoAcceptRequestEnableFileStatus)
    if os.IsNotExist(err) {
        file, err := os.Create(checkAutoAcceptRequestEnableFileStatus)
        if err != nil {
            log.Println(helpers.WrapErr(err, "Cant create file checkAutoAcceptRequestEnable"))
            return false
        }
        defer file.Close()
        value := false
        fmt.Fprintf(file, "%t", value)
        return value
    } else {
        file, err := os.Open(checkAutoAcceptRequestEnableFileStatus)
        if err != nil {
            log.Println(helpers.WrapErr(err, "Cant open file checkAutoAcceptRequestEnable"))
            return false
        }
        defer file.Close()
        var value bool
        fmt.Fscanf(file, "%t", &value)
        return value
    }
}

func (h* Handler) Fetch(limit int) ([]events.Event, error) {
    updates, err := h.client.GetUpdate(h.offset, limit)
    if err != nil {
        return nil, helpers.WrapErr(err, "events Fetcher error get updates")
    }
    if len(updates) == 0 {
        return nil, nil
    }
    var result []events.Event
    for _, update := range updates {
        result = append(result, makeEventFromUpdate(update))
    }

    h.offset = updates[len(updates) - 1].Id + 1
    return result, nil
}

func(h* Handler) FetchDelayedRequestsToJoin(autoAccept bool) ([]events.Event, error) {
    jsonEvents, err := h.storage.GetEventsWithDelayedMsgAfterRequestToJoin(context.TODO(), autoAccept)
    if err != nil {
        return nil, err
    }
    var allEvents []events.Event
    for _, strEvent := range jsonEvents {
        var delayEvent DelayedRequest
        err = json.Unmarshal([]byte(strEvent), &delayEvent)
        if err != nil {
            return allEvents, err
        }
        allEvents = append(allEvents, events.Event{
            Type: delayEvent.Type,
            Text: delayEvent.Text,
            Meta: &telegram.ChatJoinRequest{
                User: delayEvent.Meta.User,
                Chat: delayEvent.Meta.Chat,
            },
        })
    }

    return allEvents, nil
}

func(h* Handler) CheckDelayedMessageSendToAll() {
    message, err := h.storage.GetMessageForSend(context.TODO(), storage.KeyAllMessage)
    if message.MessageId <= 0 || message.TimeToSent.Unix() == 0 {
        return
    }
    if err != nil {
        log.Println(helpers.WrapErr(err, "Cant get message for CheckDelayedMessageSendToAll"))
    }
    err = h.sendMessageForAllUsers(message.FromChatId, message.MessageId)
    if err != nil {
        log.Println(helpers.WrapErr(err, "sendMessageForAllUsers from CheckDelayedMessageSendToAll"))
    }
}

func(h* Handler) CheckLeavers() {
    users, err := h.storage.GetAllUsers(context.TODO())
    if err != nil {
        log.Println(err)
    }
    for _, user :=range users {
        err = h.UpdateUsersActiveChannels(user)
        if err != nil {
            log.Println(err)
        }
    }
}

func(h* Handler) deleteDelayedRequestsToJoin(event events.Event) error {
    eventData, err := json.Marshal(event)
    if err != nil {
        return helpers.WrapErr(err, "DeleteDelayedRequestsToJoin: cant marshal event")
    }
    return h.storage.DeleteDelayedEventRequestToJoin(context.TODO(), eventData)
}

func (h* Handler) Process(event events.Event) error {
    switch event.Type {
    case events.Message:
        return h.processMessage(event)
    case events.CallbackQuery:
        return h.processCallBack(event)
    case events.RequestToJoin:
        return h.processRequestToJoin(event)
    default:
        return fmt.Errorf("cant Process type")
    }
}

func (h* Handler) processMessage(event events.Event) error {
    err := h.doCmd(event.Meta.(*telegram.Message))
    return helpers.WrapErr(err, "cant processMessage")
}

func (h* Handler) processCallBack(event events.Event) error {
    err := h.answerCallbackQuery(event.Meta.(*telegram.CallbackQuery))
    return helpers.WrapErr(err, "cant processMessage")
}

func (h* Handler) processRequestToJoin(event events.Event) error {
    delay, _ := h.storage.GetDelays(context.TODO(), storage.KeyDelayReqeustToJoin)
    if h.autoAcceptRequestEnable {
        ok, err := h.saveUsersIntoDbAndApproveRequestToJoin(event)
        if err != nil {
            return err
        }
        if ok && delay == 0 {
            return h.SentMessageToUserAfterAcceptRequestJoin(event)
        }
    }

    if delay > 0 || !h.autoAcceptRequestEnable {
        return h.SaveDelayedRequestsToJoin(event, delay)
    }

    return nil
}

func (h* Handler) SaveDelayedRequestsToJoin(event events.Event, delay int) error {
    eventData, err := json.Marshal(event)
    if err != nil {
        return helpers.WrapErr(err, "SaveDelayedRequestsToJoin: cant marshal event")
    }
    return h.storage.SaveDelayedEventRequestToJoin(context.TODO(), eventData, delay, h.autoAcceptRequestEnable)
}

func (h* Handler) saveUsersIntoDbAndApproveRequestToJoin(event events.Event) (bool, error) {
    userId := event.Meta.(*telegram.ChatJoinRequest).User.Id
    userExists, err := h.storage.IsUserExists(context.TODO(), userId)
    if err != nil {
        return false, helpers.WrapErr(err, "Cant check IsUserExists from saveUsersIntoDbAndApproveRequestToJoin()")
    }
    if userExists {
        err = h.storage.UpdateUser(
            context.TODO(),
            event.Meta.(*telegram.ChatJoinRequest).User.FirstName,
            event.Meta.(*telegram.ChatJoinRequest).User.LastName,
            event.Meta.(*telegram.ChatJoinRequest).User.Username,
            strconv.Itoa(event.Meta.(*telegram.ChatJoinRequest).Chat.Id),
            event.Meta.(*telegram.ChatJoinRequest).User.Id,
        )
        if err != nil {
            return false, helpers.WrapErr(err, "Cant UpdateUser with id: " + strconv.Itoa(userId))
        }

        ok, err := h.client.ApproveChatJoinRequest(userId, event.Meta.(*telegram.ChatJoinRequest).Chat.Id)
        if err != nil {
            return ok, helpers.WrapErr(
                err,
                "ApproveChatJoinRequest error from saveUsersIntoDbAndApproveRequestToJoin userId: " + strconv.Itoa(userId),
            )
        }
        return ok, nil
    }

    err = h.storage.SaveUser(
        context.TODO(),
        event.Meta.(*telegram.ChatJoinRequest).User.FirstName,
        event.Meta.(*telegram.ChatJoinRequest).User.LastName,
        event.Meta.(*telegram.ChatJoinRequest).User.Username,
        strconv.Itoa(event.Meta.(*telegram.ChatJoinRequest).Chat.Id),
        event.Meta.(*telegram.ChatJoinRequest).User.Id,
    )
    if err != nil {
        return false, helpers.WrapErr(err, "Cant SaveUser with id: " + strconv.Itoa(userId))
    }

    ok, err := h.client.ApproveChatJoinRequest(userId, event.Meta.(*telegram.ChatJoinRequest).Chat.Id)
    if err != nil {
        return ok, helpers.WrapErr(
            err,
            "ApproveChatJoinRequest error from saveUsersIntoDbAndApproveRequestToJoin userId: " + strconv.Itoa(userId),
        )
    }
    return ok, nil
}

func (h* Handler) SentMessageToUserAfterAcceptRequestJoin(event events.Event) error {
    if h.countRequests % 20 == 0 && h.countRequests != 0 {
        // telegram limits
        time.Sleep(2 * time.Second)
    }
    userId := event.Meta.(*telegram.ChatJoinRequest).User.Id
    message, err := h.storage.GetCurrentMessage(context.TODO(), storage.KeyRequestMessage)
    userExists, err := h.storage.IsUserExists(context.TODO(), userId)
    if err != nil || !userExists {
        return helpers.WrapErr(err, "Cant check IsUserExists from SentMessageToUserAfterAcceptRequestJoin()")
    }
    user, err := h.storage.GetUser(context.TODO(), userId)

    err = h.deleteDelayedRequestsToJoin(event)
    if err != nil {
        log.Println(helpers.WrapErr(err, "cant delete RequestToJoin event"))
    }
    // skip if user is chatmember
    if len(user.ChannelsIds) > 1 {
        return nil
    }
    h.countRequests ++

    err = helpers.WrapErr(
        h.client.ForwardMessage(
            userId, message.FromChatId, message.MessageId), "cant send msg user:" +  strconv.Itoa(userId),
        )
    if err != nil {
        return err
    }

    return h.storage.UpdateUser(
        context.TODO(),
        event.Meta.(*telegram.ChatJoinRequest).User.FirstName,
        event.Meta.(*telegram.ChatJoinRequest).User.LastName,
        event.Meta.(*telegram.ChatJoinRequest).User.Username,
        strconv.Itoa(event.Meta.(*telegram.ChatJoinRequest).Chat.Id),
        event.Meta.(*telegram.ChatJoinRequest).User.Id,
    )
}

func makeEventFromUpdate(update telegram.Update) events.Event {
    updateType := getEventType(update)
    event := events.Event{
        Type: updateType,
        Text: getUpdateMessage(update),
        Meta: events.Unknown,
    }
    if updateType == events.RequestToJoin {
        event.Meta = update.JoinRequest
    }
    if updateType == events.CallbackQuery {
        event.Meta = update.CallbackQuery
    }
    if updateType == events.Message {
        event.Meta = update.Message
    }
    return event
}

func getUpdateMessage(update telegram.Update) string {
    if update.Message == nil {
        return ""
    }
    return update.Message.Text
}

func getEventType(update telegram.Update) string {
    if update.JoinRequest != nil && update.JoinRequest.User.Id > 0 {
        return events.RequestToJoin
    }
    if update.CallbackQuery != nil {
        return events.CallbackQuery
    }
    if update.Message != nil {
        return events.Message
    }
    return events.Unknown
}