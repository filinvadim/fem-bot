package main

import (
	"database/sql"
	"fmt"
	api "github.com/Syfaro/telegram-bot-api"
	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
	"time"
)

var newbiesGreeting = `Новички, вступившие в чат, должны представиться — рассказать, 
откуда узнали про чат, прислать свое фото с кратким рассказом о себе и тегом #знакомство и 
укажите город ОДНИМ СООБЩЕНИЕМ ВМЕСТЕ С ФОТО. Да, это обязательно для всех и это не обсуждается.
Наше комьюнити состоит из открытых друг другу людей, которые делятся своими фото, 
общаются на откровенные и личные темы. Мы хотим комфортной коммуникации с людьми, 
которым доверяем. Доверять человеку с именем, фотографией и кратким описанием 
самой себя намного проще и приятнее, нежели картинке из интернета и псевдониму. 
Если вы присылаете знакомство, пожалуйста, оформляйте его строго в соответствии с правилом. 
Никаких "фото на аватарке, кому надо, тот посмотрит". Прислать фото для галочки, 
а потом удалять его не советуем - чат регулярно чистится от участников, 
не исполнивших это правило вне зависимости от того, отправляли они знакомство или нет.  
Если вы по каким-то причинам не желаете раскрывать свою личность, 
мы не можем вам предложить ничего, кроме как покинуть чат. 
Пожалуйста, не тратьте свое и наше время на споры, они ни к чему не приведут.`

const vacancyTag = "#vacancy"

func main() {
	db, err := newDB(os.Getenv("DB_PATH"))
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Fatal("db init error: ", err)
		}
	}
	defer db.Close()

	bot, err := api.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal("bot init error: ", err)
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := api.NewUpdate(0)
	u.Timeout = 60

	updatesChan, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updatesChan {
		if update.Message == nil {
			continue
		}
		log.Printf("New message: %#v", update.Message)

		var text string
		if update.Message.NewChatMembers != nil {
			var newbies = make([]string, 0)
			for _, mem := range *update.Message.NewChatMembers {
				newbies = append(newbies, "@"+mem.UserName)
			}
			text = fmt.Sprintf("Привет %s! %s\n", strings.Join(newbies, ","), newbiesGreeting)

		}
		if update.Message.LeftChatMember != nil {
			text = fmt.Sprintf("%s покинул/покинула чат.\n", update.Message.LeftChatMember.UserName)
		}
		if text != "" {
			msg := api.NewMessage(update.Message.Chat.ID, text)
			bot.Send(msg)
		}

		if strings.Contains(update.Message.Text, vacancyTag) ||
			strings.Contains(update.Message.Caption, vacancyTag) {

			if update.Message.Photo == nil {
				msg := api.NewMessage(update.Message.Chat.ID, "знакомства без фото не допускаются")
				bot.Send(msg)
				continue
			}
			if len(*update.Message.Photo) == 0 {
				msg := api.NewMessage(update.Message.Chat.ID, "знакомства без фото не допускаются")
				bot.Send(msg)
				continue
			}
			msgText := update.Message.Text
			if msgText == "" {
				msgText = update.Message.Caption
			}
			msgText = strings.Replace(msgText, vacancyTag, "", 1)

			if err := db.Insert(
				update.Message.From.UserName,
				(*update.Message.Photo)[0].FileID,
				msgText,
			); err != nil {
				color.New(color.FgRed).Println(err)
			}
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		msg := api.NewMessage(update.Message.Chat.ID, "")
		switch update.Message.Command() {
		case "help":
			msg.Text = "I understand /sayhi and /status."
		case "sayhi":
			msg.Text = "Hi :)"
		case "status":
			msg.Text = "I'm ok."
		case "singles":
			singles, err := db.GetAll()
			if err != nil {
				color.New(color.FgRed).Println(err)
				msg := api.NewMessage(update.Message.Chat.ID, "что-то пошло не так")
				bot.Send(msg)
				continue
			}
			if len(singles) == 0 {
				msg := api.NewMessage(update.Message.Chat.ID, "пусто")
				bot.Send(msg)
			}
			for _, s := range singles {
				photoMsg := api.NewPhotoShare(update.Message.Chat.ID, s.photoId)
				bot.Send(photoMsg)
				textMsg := api.NewMessage(update.Message.Chat.ID, s.msg)
				bot.Send(textMsg)
			}
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

const singlesDDL = `
 CREATE TABLE singles (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username VARCHAR(64) NULL UNIQUE,
        photo_id VARCHAR(64) NULL,
        description VARCHAR(64) NULL,
        created_at DATE NULL
    );`

type femDatabase struct {
	db *sql.DB
}

func newDB(path string) (*femDatabase, error) {
	if path == "" {
		path = "./fem.db"
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(singlesDDL)
	return &femDatabase{db: db}, err
}

func (fdb *femDatabase) Insert(username, photoId, desc string) (err error) {
	_, err = fdb.db.Exec(
		`INSERT INTO singles(username, description, photo_id, created_at) VALUES(?,?,?,?)
			ON CONFLICT(username) DO UPDATE SET description=EXCLUDED.description, photo_id=EXCLUDED.photo_id`,
		"@"+username,
		desc,
		photoId,
		time.Now().UTC(),
	)
	return err
}

type single struct {
	msg     string
	photoId string
}

func (fdb *femDatabase) GetAll() ([]single, error) {
	singles := make([]single, 0)

	rows, err := fdb.db.Query("SELECT * FROM singles")
	var (
		id        int
		username  string
		photoId   string
		desc      string
		createdAt time.Time
	)

	for rows.Next() {
		err = rows.Scan(&id, &username, &photoId, &desc, &createdAt)
		if err != nil {
			color.New(color.FgRed).Println(err)
			continue
		}
		singles = append(singles, single{
			msg:     username + "\n" + desc,
			photoId: photoId,
		})
	}

	return singles, rows.Close()
}

func (fdb *femDatabase) Close() (err error) {
	return fdb.db.Close()
}
