package main

import (
    "context"
    "flag"
    "log"
    "os"
    "time"
    tgClient "user-handler-bot/clients/telegram"
    event "user-handler-bot/events/telegram"
    "user-handler-bot/listener/event-telegram-listener"
    "user-handler-bot/storage/sqlite"
    "github.com/joho/godotenv"
)

const (
    sqliteStoragePath = "./storage.db"
    batchSize         = 10
)

func main () {
    token := flag.String(
        "tg-token",
        "",
        "token for access to telegram bot",
    )
    flag.Parse()

    if *token == "" {
        log.Fatal("tg-token not found")
    }

    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Get Envs error: %s", err)
    }
    host := checkRequiredEnv("API_HOST")
    admins := checkRequiredEnv("ID_BOT_ADMINS")

    setLogFormat()

    tgClient := tgClient.New(host, *token, admins)
    storage, err := sqlite.New(sqliteStoragePath)
    if err != nil {
        log.Fatal("can't connect to storage: ", err)
    }

    if err := storage.InitDbTables(context.TODO()); err != nil {
        log.Fatal("can't init storage: ", err)
    }

    event := event.New(&tgClient, storage)

    listener := event_telegram_listener.New(
        event,
        event,
        batchSize,
    )

    if err := listener.Start(); err != nil {
        log.Fatal("bot is stopped", err)
    }
}

func setLogFormat() {
    log.SetPrefix("[" + time.Now().Format("2006-01-02 15:04:05") + "] ")
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func checkRequiredEnv(name string) string {
    val := os.Getenv(name)
    if os.Getenv(name) == "" {
        log.Fatal("missing required env: ", name)
    }
    return val
}