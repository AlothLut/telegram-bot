package events

type Fetcher interface {
    Fetch(limit int) ([]Event, error)
    FetchDelayedRequestsToJoin(autoAccept bool) ([]Event, error)
    CheckDelayedMessageSendToAll()
    CheckLeavers()
}

type Processor interface {
    Process(e Event) error
    SentMessageToUserAfterAcceptRequestJoin(e Event) error
}


const (
    Unknown        = "unknown"
    Message        = "message"
    RequestToJoin  = "request_to_join"
    CallbackQuery  = "callback_query"
)

type Event struct {
    Type string
    Text string
    Meta interface{}
}