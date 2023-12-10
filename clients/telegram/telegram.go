package telegram

import (
    "bytes"
    "encoding/json"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "path"
    "strconv"
    "strings"
    "user-handler-bot/helpers"
)

type Client struct {
    host                     string
    botEndpoint              string
    client                   http.Client
    AdminsId                 []int
    lastInlineKeyBoardId     int
    lastInlineKeyBoardChatId int
}

func New(host string, botToken string, admins string) Client {
    return Client{
        host:   host,
        botEndpoint: getBotEndpoint(botToken),
        client: http.Client{},
        AdminsId: getAdminsIds(admins),
    }
}

func getAdminsIds(admins string) []int {
    var res []int
    for _, id := range strings.Split(admins, ",") {
        id, err := strconv.Atoi(id)
        if err != nil {
            log.Fatal("cant get admins id: ", err)
        }
        res = append(res, id)
    }
    return res
}

func getBotEndpoint(token string) string {
    return "bot" + token;
}

type UpdatesResponse struct {
    Ok      bool     `json:"ok"`
    Result  []Update `json:"result"`
}

type SendMessageResponse struct {
    Ok      bool     `json:"ok"`
    Message  Message `json:"result"`
}

type Message struct {
    Text    string `json:"text"`
    From    User   `json:"from"`
    Chat    Chat   `json:"chat"`
    Id      int    `json:"message_id"`
}

type Chat struct {
    Id  int `json:"id"`
}

type Update struct {
    Id             int              `json:"update_id"`
    Message        *Message         `json:"message"`
    JoinRequest    *ChatJoinRequest `json:"chat_join_request"`
    CallbackQuery  *CallbackQuery   `json:"callback_query"`
}

type ChatJoinRequest struct {
    User    User   `json:"from"`
    Chat    Chat   `json:"chat"`
}

type User struct {
    Id        int    `json:"id"`
    Channels  string
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Username  string `json:"username"`
}

type ChatMember struct {
    Ok      bool             `json:"ok"`
    Result  ChatMemberMember `json:"result"`
}

type ChatMemberMember struct {
    User    User   `json:"user"`
    Status  string `json:"status"`
}

type Approve struct {
    Result  bool `json:"result"`
}

type InlineKeyboardButton struct {
    Text         string `json:"text"`
    CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
    InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
    ChatID      int                   `json:"chat_id"`
    Text        string                `json:"text"`
    ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
    MessageId   int                   `json:"message_id,omitempty"`
}

type CallbackQuery struct {
    Data    string  `json:"data"`
    User    User    `json:"from"`
    Message Message `json:"message"`
}


func (c *Client) GetUpdate(offset int, limit int) (updates []Update, err error) {
    query := url.Values{}
    allowedUpdates, _ := json.Marshal([]string{"message", "callback_query", "chat_join_request"})
    query.Add("offset", strconv.Itoa(offset))
    query.Add("limit", strconv.Itoa(limit))
    query.Add("allowed_updates", string(allowedUpdates))
    data, err := c.doGetRequest("getUpdates", query)
    if err != nil {
        return nil, helpers.WrapErr(err, "Telegram API getUpdate error")
    }
    var result UpdatesResponse
    if err := json.Unmarshal(data, &result); err != nil {
        return  nil, helpers.WrapErr(err, "getUpdate Unmarshal error")
    }
    return result.Result, nil
}

func (c *Client) GetChatMember(userId int, chatId int) (user ChatMemberMember, err error) {
    query := url.Values{}
    query.Add("chat_id", strconv.Itoa(chatId))
    query.Add("user_id", strconv.Itoa(userId))
    data, err := c.doGetRequest("getChatMember", query)
    if err != nil {
        return ChatMemberMember{}, helpers.WrapErr(err, "Telegram API getChatMember error")
    }
    var result ChatMember
    if err := json.Unmarshal(data, &result); err != nil {
        return  ChatMemberMember{}, helpers.WrapErr(err, "getChatMember Unmarshal error")
    }
    return result.Result, nil
}

func (c *Client) ApproveChatJoinRequest(userId int, chatId int) (ok bool, err error) {
    query := url.Values{}
    query.Add("chat_id", strconv.Itoa(chatId))
    query.Add("user_id", strconv.Itoa(userId))
    data, err := c.doGetRequest("approveChatJoinRequest", query)
    if err != nil {
        return false, helpers.WrapErr(err, "Telegram API approveChatJoinRequest error")
    }
    var result Approve
    if err := json.Unmarshal(data, &result); err != nil {
        return  false, helpers.WrapErr(err, "approveChatJoinRequest Unmarshal error")
    }
    return result.Result, nil
}

func (c *Client) SendMessage(chatId int, text string) error {
    query := url.Values{}
    query.Add("chat_id", strconv.Itoa(chatId))
    query.Add("text", text)

    _, err := c.doGetRequest("sendMessage", query)

    return helpers.WrapErr(err, "sendMessage error")
}

func (c *Client) ForwardMessage(chatId int, from_chat_id int, forwardMsgId int) error {
    query := url.Values{}
    query.Add("chat_id", strconv.Itoa(chatId))
    query.Add("message_id", strconv.Itoa(forwardMsgId))
    query.Add("from_chat_id", strconv.Itoa(from_chat_id))
    query.Add("disable_notification", "")

    res, err := c.doGetRequest("copyMessage", query)
    //TODO если фиксится отправка убрать лог
    log.Println(string(res[:]))

    return helpers.WrapErr(err, "ForwardMessage error")
}

func (c *Client) SendInlineKeyBoard(msg SendMessageRequest) error {
    jsonData, err := json.Marshal(msg)
    if err != nil {
        return helpers.WrapErr(err, "SendInlineKeyBoard json.Marshal")
    }
    if c.lastInlineKeyBoardId > 0 && c.lastInlineKeyBoardChatId > 0 {
        c.DeleteMessage(c.lastInlineKeyBoardId, c.lastInlineKeyBoardChatId)
    }
    return c.sendInlineKeyBoard("sendMessage", jsonData)
}

func (c *Client) UpdateInlineKeyBoard(msg SendMessageRequest) error {
    jsonData, err := json.Marshal(msg)
    if err != nil {
        return helpers.WrapErr(err, "UpdateInlineKeyBoard json.Marshal")
    }
    return c.sendInlineKeyBoard("editMessageText", jsonData)
}

func (c *Client) sendInlineKeyBoard(method string, data []byte) error {
    query := url.Values{}
    query.Encode()
    requestUrl := url.URL{
        Scheme: "https",
        Host: c.host,
        Path: path.Join(c.botEndpoint, method),
    }
    resp, err := http.Post(
        requestUrl.String(),
        "application/json",
        bytes.NewBuffer(data),
    )
    if err != nil {
        log.Println(err)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    var result SendMessageResponse
    if err := json.Unmarshal(body, &result); err != nil {
        log.Println(err)
    }
    if err == nil && result.Message.Id > 0 && result.Message.Chat.Id > 0 {
        c.lastInlineKeyBoardId = result.Message.Id
        c.lastInlineKeyBoardChatId = result.Message.Chat.Id
    }

    return helpers.WrapErr(err, "doPostRequest error")
}

func (c *Client) DeleteMessage(messageId int, chatId int) error {
    query := url.Values{}
    query.Add("chat_id", strconv.Itoa(chatId))
    query.Add("message_id", strconv.Itoa(messageId))

    _, err := c.doGetRequest("deleteMessage", query)

    return helpers.WrapErr(err, "deleteMessage error")
}

func(c *Client) doGetRequest(methodName string, query url.Values) ([]byte, error) {
    const errMsg = "get request error"
    requestUrl := url.URL{
        Scheme: "https",
        Host: c.host,
        Path: path.Join(c.botEndpoint, methodName),
    }
    request, err := http.NewRequest(http.MethodGet, requestUrl.String(), nil)
    if err != nil {
        return nil, helpers.WrapErr(err, errMsg)
    }

    request.URL.RawQuery = query.Encode()
    response, err := c.client.Do(request)
    if err != nil {
        return nil, helpers.WrapErr(err, errMsg)
    }
    defer response.Body.Close()
    body, err := io.ReadAll(response.Body)
    if err != nil {
        return nil, helpers.WrapErr(err, errMsg)
    }

    return body, nil
}