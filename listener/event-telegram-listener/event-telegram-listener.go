package event_telegram_listener

import (
    "log"
    "time"
    "user-handler-bot/events"
    "user-handler-bot/helpers"
)

type Listener struct {
    fetcher     events.Fetcher
    processor   events.Processor
    eventLimit  int
    lostEvents  []events.Event
}

func New (fetcher events.Fetcher, processor events.Processor, eventLimit int) Listener {
    return Listener{
        fetcher: fetcher,
        processor: processor,
        eventLimit: eventLimit,
    }
}

func (l *Listener) Start() error {
    log.Println("App started")
    //handle delayed request_to_join
    go l.processDelayedSentMsgAfterRequestToJoin()
    go l.processSendMessageToAllUsers()
    go l.processCheckLeavers()
    for {
        events, err := l.fetcher.Fetch(l.eventLimit)
        if err != nil {
            log.Println(helpers.WrapErr(err, "Cant fetch events in Start() from listener"))
            continue
        }

        if len(events) == 0 {
            time.Sleep(3 * time.Second)
            continue
        }

        if err := l.handleEvents(events); err != nil {
            log.Println(helpers.WrapErr(err, "Cant handleEvents from Start"))
            continue
        }


        // handle lost events:
        if  len(l.lostEvents) != 0 && len(l.lostEvents) % 30 == 0 {
            err := l.handleEvents(l.lostEvents)
            if err != nil {
                log.Println(helpers.WrapErr(err, "Cant handleEvents lost events from Start"))
            }
        }
        if len(l.lostEvents) == 100 {
            log.Println("many unhandled events")
            l.lostEvents = nil
            //TODO: mb save
        }
    }
}

func (l *Listener) processDelayedSentMsgAfterRequestToJoin() {
    log.Println("start processDelayedSentMsgAfterRequestToJoin")
    for {
        withAcceptRequestToJoin := true
        delayedRequestsToJoin, err := l.fetcher.FetchDelayedRequestsToJoin(withAcceptRequestToJoin)
        if err != nil {
            log.Println(helpers.WrapErr(err, "FetchDelayedRequestsToJoin error"))
            continue
        }
        for _, requestToJoinEvent := range delayedRequestsToJoin {
            err = l.processor.SentMessageToUserAfterAcceptRequestJoin(requestToJoinEvent)
            if err != nil {
                log.Println(helpers.WrapErr(err, "cant sent message processDelayedRequestToJoin"))
                continue
            }
        }
        time.Sleep(3 * time.Second)
    }
}

func (l *Listener) processSendMessageToAllUsers() {
    log.Println("start CheckDelayedMessageSendToAll")
    for {
        l.fetcher.CheckDelayedMessageSendToAll()
    }
}


func (l *Listener) processCheckLeavers() {
    log.Println("start procesCheckLeavers")
    for {
        time.Sleep(5 * time.Second)
        l.fetcher.CheckLeavers()
        time.Sleep(20 * time.Minute)
    }
}


func (l *Listener) handleEvents(gotEvents []events.Event) error {
    for _, event := range gotEvents {
        isNewLostEvent := true
        var processErr error
        for i := 0; i < 3; i++ {
            processErr = l.processor.Process(event)
            if processErr != nil {
                log.Println(helpers.WrapErr(processErr, "cant handle event"))
                time.Sleep(3 * time.Second)
            } else {
                break
            }
        }

        // remove lost event if handled
        for i, lostEvent := range l.lostEvents {
            if lostEvent == event && processErr == nil {
                l.lostEvents = append(l.lostEvents[:i], l.lostEvents[i+1:]...)
                break
            } else if lostEvent == event && processErr != nil {
                isNewLostEvent = false
                break
            }
        }


        if processErr != nil {
            if event.Type == events.RequestToJoin && isNewLostEvent {
                log.Println(helpers.WrapErr(processErr, "cant handle RequestToJoin event after 3 tries"))
                l.lostEvents = append(l.lostEvents, event)
            }
        }
    }
    return nil
}