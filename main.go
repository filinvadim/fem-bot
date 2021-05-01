package main

import (
	"fmt"
	api "github.com/Syfaro/telegram-bot-api"
	"log"
	"os"
	"strings"
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

func main() {
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
	}
}
