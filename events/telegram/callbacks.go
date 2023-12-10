package telegram

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "time"
    "user-handler-bot/clients/telegram"
    "user-handler-bot/helpers"
    "user-handler-bot/messages"
    "user-handler-bot/storage"
)

const (
    SetSendMsg   = "/setSendingMessage"
    ShowSendMsg = "/showSendingMessage"
    SetRequestMsg = "/setRequestMessage"
    ShowRequestMsg = "/showRequestMessage"
    RequestToJoin = "/requestToJoin"
    SetTimeForSentMessageToAllUsers = "/setTimeForSentMessageToAllUsers"
    Statistics = "/stat"
    GetBack = "/get-back"
    InitSetDelay = "/init-set-delay"
    SetDelay = "/set-delay"
    CheckNotAcceptedUsers = "/check-not-accepted-users"
    ApproveNotAcceptedUsers = "/approve-not-accepted-users"
)

func (h* Handler) answerCallbackQuery(callback *telegram.CallbackQuery) error {
    if !h.isAdmin(callback.User.Id) {
        return h.client.SendMessage(callback.Message.Chat.Id, messages.ACESS_DENIED)
    }
    chatId := callback.Message.Chat.Id
    command := callback.Data
    messageId := callback.Message.Id
    h.lastInlineKeyBoardId = callback.Message.Id

    if strings.HasPrefix(command, SetDelay) {
        if err := h.checkDelay(command, storage.KeyDelayReqeustToJoin); err != nil {
            return helpers.WrapErr(err, "cant checkDelay")
        }
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.KEYBOARD_ACCEPTANCE_DELAY, h.getDelayRequestToJoinInlineKeyBoard()),
        )
    }

    switch command {
    case CheckNotAcceptedUsers:
        statusAcceptedWas := false
        notAcceptedUsers, _ := h.FetchDelayedRequestsToJoin(statusAcceptedWas)
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(
                chatId,
                messageId,
                fmt.Sprintf(messages.NOT_ACCEPTED_USERS + strconv.Itoa(len(notAcceptedUsers))),
                h.getNotAcceptedUsersInlineKeyBoard(),
            ),
        )
    case ApproveNotAcceptedUsers:
        go h.proccessAcceptMissingUsers()
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.START_ACCEPT_USERS, h.getBaseInlineKeyBoard()),
        )
    case SetTimeForSentMessageToAllUsers:
        h.setNewTimeForSentMessageToAll = true
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.SET_TIME_FOR_SENDING_MESSAGE, h.getBackToStartInlineKeyBoard()),
        )
    case InitSetDelay:
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.KEYBOARD_ACCEPTANCE_DELAY, h.getDelayRequestToJoinInlineKeyBoard()),
        )
    case GetBack:
        h.setNewTimeForSentMessageToAll = false
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.LIST_OF_COMMANDS, h.getBaseInlineKeyBoard()),
        )
    case SetSendMsg:
        h.nextSetSendMsg = storage.KeyAllMessage
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.SET_SENDING_MESSAGE, h.getBaseInlineKeyBoard()),
        )
    case SetRequestMsg:
        h.nextSetSendMsg = storage.KeyRequestMessage
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.SET_REQUEST_TO_JOIN_MESSAGE, h.getBaseInlineKeyBoard()),
        )
    case ShowSendMsg:
        err := h.showCurrentMessage(chatId, storage.KeyAllMessage)
        if err != nil {
            return err
        }
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.KEYBOARD_THIS_IS_MSG_TO_SEND, h.getBackToStartInlineKeyBoard()),
        )
    case ShowRequestMsg:
        err := h.showCurrentMessage(chatId, storage.KeyRequestMessage)
        if err != nil {
            return err
        }
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.KEYBOARD_THIS_IS_MSG_TO_REQUEST_TO_JOIN, h.getBackToStartInlineKeyBoard()),
        )
    case Statistics:
        return h.sendStat(chatId, messageId)
    case RequestToJoin:
        if h.autoAcceptRequestEnable {
            h.autoAcceptRequestEnable = false
        } else {
            h.autoAcceptRequestEnable = true
        }
        h.setAutoAcceptRequestStatusToFile()
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.LIST_OF_COMMANDS, h.getBaseInlineKeyBoard()),
        )
    default:
        return h.client.SendMessage(chatId, "Command not found")
    }
}

func (h* Handler) proccessAcceptMissingUsers () {
    statusAcceptedWas := false
    notAcceptedUsers, _ := h.FetchDelayedRequestsToJoin(statusAcceptedWas)
    for _, event := range notAcceptedUsers {
        _, err := h.saveUsersIntoDbAndApproveRequestToJoin(event)
        if err != nil {
            log.Println(err)
        }
        err = h.SentMessageToUserAfterAcceptRequestJoin(event)
        if err != nil {
            log.Println(err)
        }
    }
}

func (h* Handler) makeInlineKeyBoard(chatId int, messageId int, text string, keyBoard telegram.InlineKeyboardMarkup) telegram.SendMessageRequest {
    msg := telegram.SendMessageRequest{
        ChatID:      chatId,
        Text:        text,
        ReplyMarkup: &keyBoard,
        MessageId: messageId,
    }

    return msg
}

func (h* Handler) setAutoAcceptRequestStatusToFile() {
    file, err := os.Create(checkAutoAcceptRequestEnableFileStatus)
    if err != nil {
        log.Println(helpers.WrapErr(err, "Cant Open file setAutoAcceptRequestStatusToFile"))
        return
    }
    defer file.Close()
    _, err = file.Seek(0, 0)
    if err != nil {
        log.Println(helpers.WrapErr(err, "Cant Open file setAutoAcceptRequestStatusToFile"))
        return
    }
    _, err = fmt.Fprintf(file, "%t", h.autoAcceptRequestEnable)
    if err != nil {
        log.Println(helpers.WrapErr(err, "Cant Open file setAutoAcceptRequestStatusToFile"))
        return
    }
}

func (h* Handler) isAdmin(userId int) bool {
    found := false
    for _, id := range h.client.AdminsId {
        if id == userId {
            found = true
            break
        }
    }
    return found
}

func (h* Handler) setMsg(msgId int, chatId int, key string) error {
    return h.storage.SaveMessage(context.TODO(), msgId, chatId, key)
}

func(h* Handler) showCurrentMessage(chatId int, key string) error {
    currentMessageId, err := h.storage.GetCurrentMessage(context.TODO(), key)
    if err != nil {
        log.Println(err)
        return h.client.SendMessage(chatId, "message not found")
    }
    return h.client.ForwardMessage(chatId, currentMessageId.FromChatId, currentMessageId.MessageId)
}

func (h* Handler) sendMessageForAllUsers(chatId int, messageId int) error {
    users, err := h.storage.GetAllUsers(context.TODO())
    if err != nil {
        return helpers.WrapErr(err, "cant get users for send message")
    }
    h.processSendMessageForAllUsers(users)
    return nil
}

func (h* Handler) processSendMessageForAllUsers(users []storage.User) {
    message, err := h.storage.GetCurrentMessage(context.TODO(), storage.KeyAllMessage)
    if err != nil {
        log.Println(helpers.WrapErr(err, "Cant get message for processSendMessageForAllUsers"))
    }
    var usersIds []int
    for i, user := range users {
        if len(user.ChannelsIds) > 0 {
            err := h.UpdateUsersActiveChannels(user)
            if err != nil {
                log.Println(helpers.WrapErr(err, "cant updateUsersActiveChannels in processSendMessageForAllUsers"))
            }
        }

        if user.LastMessageId == message.MessageId {
            continue
        }

        // telegram limits
        if i % 20 == 0 && i != 0 {
            time.Sleep(3 * time.Second)
        }
        err := h.client.ForwardMessage(user.Id, message.FromChatId, message.MessageId)
        if err != nil {
            log.Println(err)
            log.Println(helpers.WrapErr(
                err, "cant send message for username:" + user.Username + 
                " user_first_name:" + user.FirstName + 
                " user_last_name:" + user.LastName +
                " user_id:" + strconv.Itoa(user.Id)))
        }
        users[i].LastMessageId = message.MessageId
        usersIds = append(usersIds, user.Id)
    }
    err = h.storage.UpdateUsersLastMessage(context.TODO(), strconv.Itoa(message.MessageId), usersIds)
    if err != nil {
        log.Println(helpers.WrapErr(err, "cant UpdateUsersLastMessage in processSendMessageForAllUsers"))
    }

    err = h.storage.DeleteMessage(context.TODO(), storage.KeyAllMessage)
    if err != nil {
        log.Println(helpers.WrapErr(err, "cant DeleteMessage after sent message to all users"))
    }
}

func (h* Handler) UpdateUsersActiveChannels(user storage.User) error {
    var leavedChannels []string
    for _, chatId := range user.ChannelsIds {
        intChatId, _ := strconv.Atoi(chatId)
        if !h.isUserChatMember(user, intChatId) {
            leavedChannels = append(leavedChannels, chatId)
        }
    }

    if len(leavedChannels) > 0 {
        // delete if already exist
        user.LeavedChannelsIds = helpers.RemoveFromSliceStringsExistsInOtherSlice(user.LeavedChannelsIds, leavedChannels)
        for _, channelsId := range leavedChannels {
            user.LeavedChannelsIds = append(user.LeavedChannelsIds, channelsId)
        }
        return h.storage.UpdateUsersLeavedChannels(context.TODO(), user)
    }
    return nil
}

func (h* Handler) sendStat(chatId int, messageId int) error {
    lastMessage, err := h.storage.GetCurrentMessage(context.TODO(), storage.KeyLastMessageAll)
    if err != nil {
        if lastMessage.MessageId == 0 {
            return h.client.UpdateInlineKeyBoard(
                h.makeInlineKeyBoard(
                    chatId,
                    messageId,
                    messages.ERR_MSG_TO_ALL_NOT_FOUND,
                    h.getBaseInlineKeyBoard(),
                ),
            )
        } else {
            return err
        }
    }

    usersCount, err := h.storage.GetCountUsers(context.TODO())
    if err != nil {
        return err
    }
    usersCountWithLastMsg, err := h.storage.GetCountUsersWithLastMsgId(context.TODO(), lastMessage.MessageId)
    if err != nil {
        return err
    }

    process := messages.USERS_NOT_FOUND
    if usersCount > 0 {
        process = messages.USERS_IN_DB + " " + strconv.Itoa(usersCount) + ". " + 
            messages.SENT + " " + strconv.Itoa((usersCountWithLastMsg * 100) / usersCount) + "%. "
    }

    msgWasSent := messages.SEND_MESSAGE_WILL_BE_SENT + lastMessage.TimeToSent.Format(LastMessageForAllFormat)
    isLastMessageWasSent := time.Now().After(lastMessage.TimeToSent)
    if !isLastMessageWasSent {
        msgWasSent = messages.MESSAGE_WAS_SENT + ": " + lastMessage.TimeToSent.Format(LastMessageForAllFormat)
    }
    if lastMessage.TimeToSent.Unix() == 0 {
        msgWasSent = messages.TIME_FOR_SENDING_NOT_FOUND
    }

    return h.client.UpdateInlineKeyBoard(
        h.makeInlineKeyBoard(chatId, messageId, process + msgWasSent, h.getBaseInlineKeyBoard()),
    )
}

func (h* Handler) isUserChatMember(user storage.User, chatId int) bool {
    member, err := h.client.GetChatMember(user.Id, chatId)
    if err != nil {
        log.Println(helpers.WrapErr(
            err, "user is not a member of the chat username:" + user.Username + 
            " user_first_name:" + user.FirstName + 
            " user_last_name:" + user.LastName +
            " user_id:" + strconv.Itoa(user.Id)))
        return false
    }

    if member.Status == "member" {
        return true
    }

    return false
}


func (h* Handler) getBaseInlineKeyBoard() telegram.InlineKeyboardMarkup {
    statusRequestToJoin := ""
    if h.autoAcceptRequestEnable {
        statusRequestToJoin = messages.KEYBOARD_ON_REQUEST_TO_JOIN
    } else {
        statusRequestToJoin = messages.KEYBOARD_OFF_REQUEST_TO_JOIN
    }
    return telegram.InlineKeyboardMarkup{
        InlineKeyboard: [][]telegram.InlineKeyboardButton{
        {
            {Text: messages.KEYBOARD_SET_MESSAGE_TO_SEND, CallbackData: SetSendMsg},
            {Text: messages.KEYBOARD_SHOW_MESSAGE_TO_SEND, CallbackData: ShowSendMsg},
        },
        {  
            {Text: messages.KEYBOARD_SET_REQUEST_MSG, CallbackData: SetRequestMsg},
            {Text: messages.KEYBOARD_SHOW_REQUEST_MSG, CallbackData: ShowRequestMsg},
        },
        {
            {Text: statusRequestToJoin, CallbackData: RequestToJoin},
        },
        {
            {Text: messages.KEYBOARD_ACCEPTANCE_DELAY, CallbackData: InitSetDelay},
        },
        {
            {Text: messages.KEYBOARD_SET_TIME_FOR_SEND_MESSAGE_FOR_ALL_USERS, CallbackData: SetTimeForSentMessageToAllUsers},
        },
        {
            {Text: messages.KEYBOARD_STATISTIC, CallbackData: Statistics},
        },
        {
            {Text: messages.KEYBOARD_CHECK_NOT_ACCEPTED_USERS, CallbackData: CheckNotAcceptedUsers},
        },
    },}
}

func (h* Handler) getBackToStartInlineKeyBoard() telegram.InlineKeyboardMarkup {
    return telegram.InlineKeyboardMarkup{
        InlineKeyboard: [][]telegram.InlineKeyboardButton{
        {
            {Text: messages.KEYBOARD_GET_BACK, CallbackData: GetBack},
        },
    },}
}

func (h* Handler) getDelayRequestToJoinInlineKeyBoard() telegram.InlineKeyboardMarkup {
    delay, _ := h.storage.GetDelays(context.TODO(), storage.KeyDelayReqeustToJoin)
    return telegram.InlineKeyboardMarkup{
        InlineKeyboard: h.getButtonsDelayRequestToJoin(delay),
    }
}

func (h* Handler) getNotAcceptedUsersInlineKeyBoard() telegram.InlineKeyboardMarkup {
    return telegram.InlineKeyboardMarkup{
        InlineKeyboard: h.getButtonsNotAcceptedUsers(),
    }
}

func (h* Handler) getButtonsNotAcceptedUsers() [][]telegram.InlineKeyboardButton {
    var result [][]telegram.InlineKeyboardButton
    result = append(result, []telegram.InlineKeyboardButton{{Text: messages.APPROVE_NOT_ACCEPTED_USERS, CallbackData: ApproveNotAcceptedUsers}})
    result = append(result, []telegram.InlineKeyboardButton{{Text: messages.KEYBOARD_GET_BACK, CallbackData: GetBack}})
    return result
}

func (h* Handler) getButtonsDelayRequestToJoin(delay int) [][]telegram.InlineKeyboardButton {
    defaultDelays := []int{0, 5, 10, 15, 30, 60, 300, 600, 900, 1800, 3600, 7200, 21600, 43200, 64800, 86400}
    var buttons []telegram.InlineKeyboardButton
    for _, seconds := range defaultDelays {
        text := ""
        callback := SetDelay + "?"
        callback = callback + strconv.Itoa(seconds)
        text = h.convertingDelayseconds(seconds)
        if seconds == delay {
            text = text + "*"
        }
        buttons = append(buttons, telegram.InlineKeyboardButton{Text: text, CallbackData: callback})
    }
    var result [][]telegram.InlineKeyboardButton
    chunkSize := 4
    for i := 0; i < len(buttons); i += chunkSize {
        end := i + chunkSize
        if end > len(buttons) {
            end = len(buttons)
        }
        result = append(result, buttons[i:end])
    }
    result = append(result, []telegram.InlineKeyboardButton{{Text: messages.KEYBOARD_GET_BACK, CallbackData: GetBack}})
    return result
}

func (h* Handler) convertingDelayseconds(seconds int) string {
    text := ""
    if seconds <= 30 {
        text = strconv.Itoa(seconds) + "sec"
    }
    if seconds > 30 && seconds < 3600 {
        text = strconv.Itoa(seconds/60) + "min"
    }
    if seconds >= 3600 {
        text = strconv.Itoa(seconds/3600) + "h"
    }
    return text
}

func (h* Handler) checkDelay(command string, key string) error {
    parts := strings.Split(command, "?")
    if len(parts) > 1 {
        command = SetDelay
        delay, err := strconv.Atoi(parts[1])
        if err == nil {
            helpers.WrapErr(err, "cant get time for SetDelay")
        }
        h.storage.UpdateDelays(context.TODO(), key, delay)
    }
    return nil
}