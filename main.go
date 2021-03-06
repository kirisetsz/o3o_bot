package main

import (
  "os"
  "log"
  "time"
  "net/http"
  "encoding/json"
  "strings"
  "encoding/hex"
  "crypto/md5"
  "math/rand"
  "github.com/tucnak/telebot"
)

type KaomojiDict struct {
  Tag   string `json:"tag"`
  Yan []string `json:"yan"`
}

var bot *telebot.Bot
var dict []KaomojiDict
var tags []string
var next map[string][]string = make(map[string][]string)

func main() {

  token := os.Getenv("BOT_API_TOKEN")
  if len(token) > 0 {
    log.Printf("Telegram Bot Token: %v\n", token)
  } else {
    log.Fatal("Please set 'BOT_API_TOKEN' from environment variable")
  }

  updateDict()

  if newBot, err := telebot.NewBot(token); err != nil {
    log.Fatal(err);
  } else {
    bot = newBot
  }

  bot.Messages = make(chan telebot.Message, 1000)
  bot.Queries = make(chan telebot.Query, 1000)

  go messages()
  go queries()

  log.Printf("Started.")

  bot.Start(1 * time.Second)
}

func updateDict() {
  log.Println("Updating kaomoji dictionary...")

  body := map[string][]KaomojiDict{}

  resp, err := http.Get("https://raw.githubusercontent.com/guo-yu/o3o/master/yan.json")

  if err != nil {
    log.Fatal(err)
  }

  decoder := json.NewDecoder(resp.Body)

  if err := decoder.Decode(&body); err != nil {
    log.Fatal(err)
  } else {
    dict = body["list"]
    tags = make([]string, len(dict))
    for i, e := range dict {
      tags[i] = e.Tag
      for _, o3o := range e.Yan {
        next[o3o] = e.Yan
      }
    }
    log.Println("Kaomoji dictionary initialized.")
  }
}

func messages() {
  for message := range bot.Messages {
    log.Println("--- new message ---")
    log.Println("from:", message.Sender)
    log.Println("text:", message.Text)
    switch text := message.Text; {
    case message.Text == "/start":
      bot.SendMessage(message.Chat, `Here is o3o bot.`, nil)
    case message.Text == "/tags":
      bot.SendMessage(message.Chat,
      "List of kaomoji tags:\n\n" + strings.Join(tags, "\n") +
      "\n\nFull list of kaomojies: https://github.com/guo-yu/o3o/blob/master/yan.json ",
      &telebot.SendOptions { DisableWebPagePreview: true })
    default:
      if yans, found := next[text]; found {
        bot.SendMessage(message.Chat, yans[rand.Intn(len(yans))], nil)
      } else {
        bot.SendMessage(message.Chat, `o3o is in panic.`, nil)
      }
    }
  }
}

type Result struct {
  Id string `json:"id"`
  Type string `json:"type"`
  Title string `json:"title"`
  Text string `json:"message_text"`
  Description string `json:"description"`
}

type KaomojiWrapper struct {
  Result Result
}

func (wrapper KaomojiWrapper) MarshalJSON() ([]byte, error) {
  r := wrapper.Result
  r.Type = "article";
  bytes, err := json.Marshal(r);
  return bytes, err
}

func queries() {
  for query := range bot.Queries {
    log.Println("--- new query ---")
    log.Println("from:", query.From)
    log.Println("text:", query.Text)

    results := make([]telebot.Result, 0, 19)
    result_guard := make(map[string]bool)

    for _, entry := range dict {
      if tag, q := entry.Tag, query.Text; strings.Contains(" " + tag, " " + q) {
        for _, y := range entry.Yan {
          if len(results) < cap(results) {
            sum := md5.Sum([]byte(y))
            result_id := string(hex.EncodeToString(sum[:]))
            if _, found := result_guard[result_id]; !found {
              results = append(results, &KaomojiWrapper {
                Result { Id: result_id, Title: y, Text: y, Description: tag },
              })
              result_guard[result_id] = true
            }
          }
        }
      }
    }

    // And finally respond to the query:
    if err := bot.Respond(query, results); err != nil {
      log.Println("ouch:", err)
    }
  }
}
