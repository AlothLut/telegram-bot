package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
	"user-handler-bot/helpers"
	"user-handler-bot/storage"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
    db *sql.DB
}

func New(path string) (*Storage, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, helpers.WrapErr(err, "cant open db")
    }

    errping := db.Ping()
    if errping != nil {
        return nil, helpers.WrapErr(errping, "cant ping db")
    }

    return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, firstName string, lastName string, username string, channelId string, id int) error {
    var channels []string
    channels = append(channels, channelId)
    channelsJson, err := json.Marshal(channels)
    if err != nil {
        return helpers.WrapErr(err, "cant insert new: cant marshal channelId")
    }

    query := `INSERT INTO users (date_create, id, first_name, last_name, username, channels) VALUES(?,?,?,?,?,?)`
    _, err = s.db.ExecContext(
        ctx,
        query,
        time.Now(),
        id,
        firstName,
        lastName,
        username,
        string(channelsJson),
    )
    if err != nil {
        return helpers.WrapErr(err, "cant insert new user")
    }
    return nil
}

func (s *Storage) GetUser(ctx context.Context, id int) (storage.User, error) {
    var user storage.User
    query := `SELECT id, date_create, first_name, last_name, username, channels, last_message_sent, leaved_channels FROM users where id = ?`
    rows, err := s.db.QueryContext(ctx, query, id)
    if err != nil {
        return user, helpers.WrapErr(err, "cant GetUser")
    }
    defer rows.Close()
    for rows.Next() {
        var id int
        var time time.Time
        var firstName string
        var lastName string
        var username string
        var channelsId []byte
        var lastMessage int
        var leavedChannels []byte
        err := rows.Scan(
            &id,
            &time,
            &firstName,
            &lastName,
            &username,
            &channelsId,
            &lastMessage,
            &leavedChannels,
        )
        var channelsIdStr []string
        var leavedChannelsStr []string
        json.Unmarshal(channelsId, &channelsIdStr)
        json.Unmarshal(leavedChannels, &leavedChannelsStr)
        user := storage.User{
            Id: id,
            Timestamp: time,
            FirstName: firstName,
            LastName: lastName,
            Username: username,
            ChannelsIds: channelsIdStr,
            LastMessageId: lastMessage,
            LeavedChannelsIds: leavedChannelsStr,
        }
        if err != nil {
            return user, helpers.WrapErr(err, "cant GetUser rows")
        }
    }
    return user, nil
}

func (s *Storage) UpdateUser(ctx context.Context, firstName string, lastName string, username string, channelId string, id int) error {
    channelsQuery := `SELECT channels from users WHERE id = ?`
    var channels string
    if err := s.db.QueryRowContext(ctx, channelsQuery, id).Scan(&channels); err != nil {
        return  helpers.WrapErr(err, "cant check channels for user with id " + strconv.Itoa(id))
    }

    var channelsIds []string
    err := json.Unmarshal([]byte(channels), &channelsIds)
    if err != nil {
        return  helpers.WrapErr(err, "cant Unmarshal channels for user with id: " + strconv.Itoa(id))
    }
    found := false
    for _, id := range channelsIds {
        if id == channelId{
            found = true
            break
        }
    }

    if !found {
        channelsIds = append(channelsIds, channelId)
    }

    channelsJson, err := json.Marshal(channelsIds)
    if err != nil {
        return helpers.WrapErr(err, "cant marshal channelsJson for user with id: " + strconv.Itoa(id))
    }

    query := `UPDATE users SET first_name = ?, last_name = ?, username = ?, channels = ? WHERE id = ?`
    _, err = s.db.ExecContext(
        ctx,
        query,
        firstName,
        lastName,
        username,
        string(channelsJson),
        id,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant update user with id " + strconv.Itoa(id))
    }
    return nil
}

func (s *Storage) UpdateUsersLeavedChannels(ctx context.Context, user storage.User) error {
    user.ChannelsIds = helpers.RemoveFromSliceStringsExistsInOtherSlice(user.ChannelsIds, user.LeavedChannelsIds)
    query := `UPDATE users SET channels = ?, leaved_channels = ? WHERE id = ?`
    channelsJson, err := json.Marshal(user.ChannelsIds)
    if err != nil {
        return err
    }
    leavedChannelsJson, err := json.Marshal(user.LeavedChannelsIds)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(
        ctx,
        query,
        string(channelsJson),
        string(leavedChannelsJson),
        user.Id,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant UpdateUsersLeavedChannels user with id " + strconv.Itoa(user.Id))
    }
    return nil
}

func (s *Storage) UpdateUsersLastMessage(ctx context.Context, value string, usersIds []int) error {
    placeholders := make([]string, len(usersIds))
    for i := range placeholders {
        placeholders[i] = "?"
    }
    placeholder :=  strings.Join(placeholders, ",")
    query := "UPDATE users SET last_message_sent = ? WHERE id IN (" + placeholder + ")"
    args := make([]interface{}, len(usersIds)+1)

    lastMessageSent, _ := strconv.Atoi(value)
    args[0] = lastMessageSent

    for i, id := range usersIds {
        args[i+1] = id
    }

    _, err := s.db.ExecContext(ctx, query, args...)
    if err != nil {
        return err
    }
    return nil
}

func (s *Storage) DeleteUser(ctx context.Context, userId int) error {
    query := `DELETE FROM users WHERE id = ?`
    _, err := s.db.ExecContext(
        ctx,
        query,
        userId,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant delete user with id " + strconv.Itoa(userId))
    }
    return nil
}

func (s *Storage) IsUserExists(ctx context.Context, userId int) (bool, error) {
    query := `SELECT COUNT(*) FROM users WHERE id = ?`
    var count int

    if err := s.db.QueryRowContext(ctx, query, userId).Scan(&count); err != nil {
        return false, helpers.WrapErr(err, "cant check user with id " + strconv.Itoa(userId))
    }

    return count > 0, nil
}

func (s *Storage) GetCountUsers(ctx context.Context) (int, error) {
    query := `SELECT COUNT(*) FROM users;`
    var count int

    if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
        return 0, helpers.WrapErr(err, "cant check COUNT users")
    }
    return count, nil
}

func (s *Storage) GetCountUsersWithLastMsgId(ctx context.Context, lastMessageId int) (int, error) {
    query := `SELECT COUNT(*) FROM users where last_message_sent = ?;`
    var count int

    if err := s.db.QueryRowContext(ctx, query, lastMessageId).Scan(&count); err != nil {
        return 0, helpers.WrapErr(err, "cant check COUNT users with last_message_sent")
    }
    return count, nil
}

func (s *Storage) GetAllUsers(ctx context.Context) ([]storage.User, error) {
    var users []storage.User
    query := `SELECT id, date_create, first_name, last_name, username, channels, COALESCE(last_message_sent, 0), leaved_channels FROM users`
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return users, helpers.WrapErr(err, "cant GetAllUsers")
    }
    defer rows.Close()
    for rows.Next() {
        var id int
        var time time.Time
        var firstName string
        var lastName string
        var username string
        var channelsId []byte
        var lastMessage int
        var leavedChannels []byte
        err := rows.Scan(
            &id,
            &time,
            &firstName,
            &lastName,
            &username,
            &channelsId,
            &lastMessage,
            &leavedChannels,
        )
        var channelsIdStr []string
        var leavedChannelsStr []string
        json.Unmarshal(channelsId, &channelsIdStr)
        json.Unmarshal(leavedChannels, &leavedChannelsStr)
        user := storage.User{
            Id: id,
            Timestamp: time,
            FirstName: firstName,
            LastName: lastName,
            Username: username,
            ChannelsIds: channelsIdStr,
            LastMessageId: lastMessage,
            LeavedChannelsIds: leavedChannelsStr,
        }
        if err != nil {
            return users, helpers.WrapErr(err, "cant GetAllUsers rows")
        }
        users = append(users, user)
    }
    return users, nil
}

func (s *Storage) SaveMessage(ctx context.Context, messageId int, chatId int, key string) error {
    remove_old_value := `DELETE FROM messages WHERE key = ?`
    _, err := s.db.ExecContext(
        ctx,
        remove_old_value,
        key,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant remove old message with key:" + key)
    }

    query := `INSERT INTO messages (key, forward_message_id, chat_id) VALUES(?, ?, ?)`
    _, err = s.db.ExecContext(
        ctx,
        query,
        key,
        messageId,
        chatId,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant insert new message with key:" + key)
    }
    return nil
}

func (s *Storage) GetCurrentMessage(ctx context.Context, key string) (storage.ForwardMessage, error) {
    var msg storage.ForwardMessage
    query := `SELECT chat_id, forward_message_id, time_for_sent FROM messages WHERE key = ?`
    if err := s.db.QueryRowContext(ctx, query, key).Scan(&msg.FromChatId, &msg.MessageId, &msg.TimeToSent); err != nil {
        return msg, helpers.WrapErr(err, "cant select text from message with key:" + key)
    }
    return msg, nil
}

func (s *Storage) SetTimeToSentForMessage(ctx context.Context, key string, date time.Time) error {
    query := `UPDATE messages SET time_for_sent = ? WHERE key = ?;`
    _, err := s.db.ExecContext(
        ctx,
        query,
        date,
        key,
    )
    if err != nil {
        return err
    }
    return nil
}

func (s *Storage) GetMessageForSend(ctx context.Context, key string) (storage.ForwardMessage, error) {
    now := time.Now()
    var msg storage.ForwardMessage
    var timeForStart string
    query := `SELECT chat_id, forward_message_id, time_for_sent FROM messages WHERE key = ? and time_for_sent <= ?`
    if err := s.db.QueryRowContext(ctx, query, key, now).Scan(&msg.FromChatId, &msg.MessageId, &timeForStart); err != nil {
        return msg, helpers.WrapErr(err, "GetMessageForSend cant select text from message with key:" + key)
    }
    if timeForStart != "" {
        t, err := time.Parse(time.RFC3339, timeForStart)
        if err != nil {
            log.Println(err)
        }
        msg.TimeToSent = t
    }
    return msg, nil
}

func (s *Storage) DeleteMessage(ctx context.Context, key string) error {
    query := `DELETE FROM messages WHERE key = ?`
    _, err := s.db.ExecContext(
        ctx,
        query,
        key,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant delete message with key:" + key)
    }
    return nil
}

func (s *Storage) UpdateDelays(ctx context.Context, key string, delay int) error {
    query := `INSERT OR REPLACE INTO delays (key, delay_seconds) VALUES (?, ?);`
    _, err := s.db.ExecContext(
        ctx,
        query,
        key,
        delay,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant delete message with key:" + key)
    }
    return nil
}

func (s *Storage) GetDelays(ctx context.Context, key string) (int, error) {
    var delay int
    query := `SELECT delay_seconds FROM delays WHERE key = ?;`
    if err := s.db.QueryRowContext(ctx, query, key).Scan(&delay); err != nil {
        return delay, helpers.WrapErr(err, "cant GetDelays key:" + key)
    }
    return delay, nil
}

func (s *Storage) SaveDelayedEventRequestToJoin(ctx context.Context, data []byte, delay int, autoAcceptStatus bool) error {
    now := time.Now()
    timeForAcceptedRequest := now.Add(time.Duration(delay) * time.Second)
    query := `INSERT OR REPLACE INTO requests_to_join (event_request_to_join, date_sent_message, auto_accept_status) VALUES (?, ?, ?);`
    _, err := s.db.ExecContext(
        ctx,
        query,
        string(data),
        timeForAcceptedRequest,
        autoAcceptStatus,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant SaveDelayedEventRequestToJoin")
    }
    return nil
}

func (s *Storage) DeleteDelayedEventRequestToJoin(ctx context.Context, data []byte) error {
    query := `DELETE FROM requests_to_join WHERE event_request_to_join = ?;`
    _, err := s.db.ExecContext(
        ctx,
        query,
        string(data),
    )
    if err != nil {
        return helpers.WrapErr(err, "DeleteDelayedEventRequestToJoin error")
    }
    return nil
}

func (s *Storage) GetEventsWithDelayedMsgAfterRequestToJoin(ctx context.Context, autoAcceptStatus bool) ([]string, error) {
    now := time.Now()
    query := `SELECT event_request_to_join from requests_to_join WHERE date_sent_message <= ? and auto_accept_status = ?`
    var eventsJson []string

    rows, err := s.db.QueryContext(ctx, query, now, autoAcceptStatus)
    if err != nil {
        return eventsJson, helpers.WrapErr(err, "cant select GetEventsWithDelayedMsgAfterRequestToJoin")
    }
    defer rows.Close()
    for rows.Next() {
        var eventJson string
        err := rows.Scan(&eventJson)
        if err != nil {
            return eventsJson, helpers.WrapErr(err, "cant GetEventsWithDelayedMsgAfterRequestToJoin rows")
        }
        eventsJson = append(eventsJson, eventJson)
    }
    return eventsJson, nil
}

func (s *Storage) InitDbTables(ctx context.Context) error {
    users := `CREATE TABLE IF NOT EXISTS users (id int not null unique, date_create timestamp default current_timestamp, 
        first_name text not null default "", last_name text not null default "", username text not null default "", 
        channels json not null default "", last_message_sent int, leaved_channels json not null default "");`
    messages := `CREATE TABLE IF NOT EXISTS messages (key text not null unique, forward_message_id int, chat_id int, time_for_sent timestamp default 0);`
    delays := `CREATE TABLE IF NOT EXISTS delays (key text not null unique, delay_seconds int);`
    requests_to_join := `CREATE TABLE IF NOT EXISTS requests_to_join (event_request_to_join json unique, date_sent_message timestamp default 0, auto_accept_status boolean default false);`
    query := users + messages + delays + requests_to_join
    _, err := s.db.ExecContext(
        ctx,
        query,
    )
    if err != nil {
        return helpers.WrapErr(err, "cant init tables for db")
    }
    return nil
}