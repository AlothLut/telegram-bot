package messages

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

var (
    KEYBOARD_SET_MESSAGE_TO_SEND = getenv("KEYBOARD_SET_MESSAGE_TO_SEND", "Set message to send")
    KEYBOARD_SHOW_MESSAGE_TO_SEND = getenv("KEYBOARD_SHOW_MESSAGE_TO_SEND", "ShowSendingMessage")
    KEYBOARD_SET_REQUEST_MSG = getenv("KEYBOARD_SET_REQUEST_MSG", "SetRequestMsg")
    KEYBOARD_SHOW_REQUEST_MSG = getenv("KEYBOARD_SHOW_REQUEST_MSG", "ShowRequestMsg")
    KEYBOARD_OFF_REQUEST_TO_JOIN= getenv("KEYBOARD_OFF_REQUEST_TO_JOIN", "Request to join: Off")
    KEYBOARD_ON_REQUEST_TO_JOIN= getenv("KEYBOARD_ON_REQUEST_TO_JOIN", "Request to join: On")
    KEYBOARD_SET_TIME_FOR_SEND_MESSAGE_FOR_ALL_USERS = getenv("KEYBOARD_SET_TIME_FOR_SEND_MESSAGE_FOR_ALL_USERS", "Set time for send message to all users")
    KEYBOARD_STATISTIC = getenv("KEYBOARD_STATISTIC", "Statistics")
    KEYBOARD_GET_BACK = getenv("KEYBOARD_GET_BACK", "get back")
    KEYBOARD_THIS_IS_MSG_TO_SEND = getenv("KEYBOARD_THIS_IS_MSG_TO_SEND", "^This is current message to send all users")
    KEYBOARD_THIS_IS_MSG_TO_REQUEST_TO_JOIN = getenv("KEYBOARD_THIS_IS_MSG_TO_REQUEST_TO_JOIN", "^This is current message to request to join")
    KEYBOARD_ACCEPTANCE_DELAY = getenv("KEYBOARD_ACCEPTANCE_DELAY", "Acceptance delay for request to join")
    KEYBOARD_CHECK_NOT_ACCEPTED_USERS = getenv("KEYBOARD_CHECK_NOT_ACCEPTED_USERS", "Show info about not accepted users")


    ACESS_DENIED = getenv("ACCESS_DENIED", "Access is denied")
    LIST_OF_COMMANDS = getenv("LIST_OF_COMMANDS", "List of commands")
    SET_MESSAGE_TO_SEND_UPDATED = getenv("SET_MESSAGE_TO_SEND_UPDATED", "Message to send was updated")
    SEND_MESSAGE_WILL_BE_SENT = getenv("SEND_MESSAGE_WILL_BE_SENT", "Message for all users will be sent: ")
    NOT_ACTIVE_SEND_MESSAGE_FOR_ALL = getenv("NOT_ACTIVE_SEND_MESSAGE_FOR_ALL", "No active mailings found")
    SET_TIME_FOR_SENDING_MESSAGE = getenv("SET_TIME_FOR_SENDING_MESSAGE", "Send message with time for sent message to all users in format: day.month.year hours:minutes like 02.01.2006 15:04")
    SET_SENDING_MESSAGE = getenv("SET_SENDING_MESSAGE", "Send a message to send to all users in this chat and do not delete it before sending")
    SET_REQUEST_TO_JOIN_MESSAGE = getenv("SET_REQUEST_TO_JOIN_MESSAGE", "Send a message that will be sent when accepted into the group and do not delete it until you change to a new one")
    APPROVE_NOT_ACCEPTED_USERS = getenv("APPROVE_NOT_ACCEPTED_USERS", "Approve not accepted users")
    NOT_ACCEPTED_USERS = getenv("NOT_ACCEPTED_USERS", "The number of unaccepted users in the database: ")
    START_ACCEPT_USERS = getenv("START_ACCEPT_USERS", "Accept users was started")

    ERR_PARSE_TIME_FOR_SENT_MSG_TO_ALL = getenv("ERR_PARSE_TIME_FOR_SENT_MSG_TO_ALL", "Can not parse this time check format, required: 02.01.2006 15:04 dd.mm.yyyy hh:mm")
    ERR_MSG_TO_ALL_NOT_FOUND = getenv("ERR_MSG_TO_ALL_NOT_FOUND", "Message to sent all users not found")

    USERS_NOT_FOUND = getenv("USERS_NOT_FOUND", "users not found ")
    SENT = getenv("SENT", "sent")
    USERS_IN_DB = getenv("USERS_IN_DB", "users in db:")
    MESSAGE_WAS_SENT = getenv("MESSAGE_WAS_SENT", "The message was sent")
    TIME_FOR_SENDING_NOT_FOUND = getenv("TIME_FOR_SENDING_NOT_FOUND", "Time for sending message is not found")
)


func getenv(key, fallback string) string {
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Get Envs error: %s", err)
    }
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}