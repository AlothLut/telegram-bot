package telegram

import (
    "context"
    "log"
    "strings"
    "time"
    "user-handler-bot/clients/telegram"
    "user-handler-bot/helpers"
    "user-handler-bot/messages"
    "user-handler-bot/storage"
)


const (
    Start = "/start"
    LastMessageForAllFormat = "02.01.2006 15:04"
)


func (h* Handler) doCmd(message *telegram.Message) error {
    user := message.From
    chatId := message.Chat.Id
    text := message.Text
    messageId := message.Id
    if !h.isAdmin(user.Id) {
        return h.client.SendMessage(chatId, messages.ACESS_DENIED)
    }
    text = strings.TrimSpace(text)

    if h.nextSetSendMsg != "" {
        keyMessage := h.nextSetSendMsg
        h.nextSetSendMsg = ""
        err := h.setMsg(messageId, chatId, keyMessage)
        if err != nil {
            return helpers.WrapErr(err, "cant set this messeage for sending")
        }

        if keyMessage == storage.KeyAllMessage {
            err = h.setMsg(messageId, chatId, storage.KeyLastMessageAll)
            if err != nil {
                return helpers.WrapErr(err, "cant set this messeage for sending")
            }
        }

        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.SET_MESSAGE_TO_SEND_UPDATED, h.getBaseInlineKeyBoard()),
        )
    }

    if h.setNewTimeForSentMessageToAll {
        h.setNewTimeForSentMessageToAll = false
        timeToRun, err := time.Parse(LastMessageForAllFormat, text)
        if err != nil {
            log.Println(helpers.WrapErr(err, "setNewTimeForSentMessageToAll parse time error"))
            return h.client.UpdateInlineKeyBoard(
                h.makeInlineKeyBoard(chatId, h.lastInlineKeyBoardId, messages.ERR_PARSE_TIME_FOR_SENT_MSG_TO_ALL, h.getBaseInlineKeyBoard()),
            )
        }

        lastMessage, err := h.storage.GetCurrentMessage(context.TODO(), storage.KeyAllMessage)
        if lastMessage.MessageId <= 0 {
            return h.client.UpdateInlineKeyBoard(
                h.makeInlineKeyBoard(
                    chatId,
                    h.lastInlineKeyBoardId,
                    messages.ERR_MSG_TO_ALL_NOT_FOUND,
                    h.getBaseInlineKeyBoard(),
                ),
            )
        }

        err = h.storage.SetTimeToSentForMessage(context.TODO(), storage.KeyAllMessage, timeToRun)
        if err != nil {
            log.Println(err)
        }
        err = h.storage.SetTimeToSentForMessage(context.TODO(), storage.KeyLastMessageAll, timeToRun)
        if err != nil {
            log.Println(err)
        }
        return h.client.UpdateInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, h.lastInlineKeyBoardId, messages.SEND_MESSAGE_WILL_BE_SENT + timeToRun.Format(LastMessageForAllFormat), h.getBaseInlineKeyBoard()),
        )
    }

    switch text {
    case Start:
        return h.client.SendInlineKeyBoard(
            h.makeInlineKeyBoard(chatId, messageId, messages.LIST_OF_COMMANDS, h.getBaseInlineKeyBoard()),
        )
    default:
        return h.client.SendMessage(chatId, "Command not found")
    }
}