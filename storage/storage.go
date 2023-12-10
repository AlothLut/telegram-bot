package storage

import (
    "context"
    "time"
)

type Storage interface {
    SaveUser(ctx context.Context, firstName string, lastName string, username string, channelId string, id int) error
    UpdateUser(ctx context.Context, firstName string, lastName string, username string, channelId string, id int) error
    UpdateUsersLastMessage(ctx context.Context, value string, usersIds []int) error
    UpdateUsersLeavedChannels(ctx context.Context, user User) error
    GetUser(ctx context.Context, userId int) (User, error)
    DeleteUser(ctx context.Context, userId int) error
    IsUserExists(ctx context.Context, userId int) (bool, error)
    GetAllUsers(ctx context.Context) ([]User, error)
    GetCountUsersWithLastMsgId(ctx context.Context, lastMessageId int) (int, error)
    GetCountUsers(ctx context.Context) (int, error)
    SaveMessage(ctx context.Context, messageId int, chatId int, key string) error
    DeleteMessage(ctx context.Context, key string) error
    GetCurrentMessage(ctx context.Context, key string) (ForwardMessage, error)
    GetMessageForSend(ctx context.Context, key string) (ForwardMessage, error)
    SetTimeToSentForMessage(ctx context.Context, key string, date time.Time) error
    UpdateDelays(ctx context.Context, key string, delay int) error
    GetDelays(ctx context.Context, key string) (int, error)
    SaveDelayedEventRequestToJoin(ctx context.Context, data []byte, delat int, autoAcceptStatus bool) error
    GetEventsWithDelayedMsgAfterRequestToJoin(ctx context.Context, autoAcceptStatus bool) ([]string, error)
    DeleteDelayedEventRequestToJoin(ctx context.Context, data []byte) error
}

type User struct {
    Timestamp     time.Time
    Id            int
    ChannelsIds   []string
    FirstName     string
    LastName      string
    Username      string
    LastMessageId int
    LeavedChannelsIds []string
}

type ForwardMessage struct {
    FromChatId  int
    MessageId   int
    TimeToSent  time.Time
}

type Message struct {
    Text    string
}

const (
    KeyRequestMessage = "request_message"
    KeyAllMessage = "message_all"
    KeyLastMessageAll = "last_message_all"
    KeyDelayReqeustToJoin = "delay_request_to_join"
)