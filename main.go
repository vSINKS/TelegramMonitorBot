package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	disk "github.com/shirou/gopsutil/disk"
	host "github.com/shirou/gopsutil/host"
	load "github.com/shirou/gopsutil/load"
	mem "github.com/shirou/gopsutil/mem"
)

/*
 * Config storage of the telegram token in order to create the bot, to create one, http://t.me/@BotFather
 */
type Config struct {
	Token string `json:"telegram_token"`
}

func main() {

	// Read config file
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config Config
	json.Unmarshal(byteValue, &config)

	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Println("Is the token set?")
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		//log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ReplyToMessageID = update.Message.MessageID

		// =========== HOST =============
		hostStat, _ := host.Info()
		msg.Text += fmt.Sprintf("====%s====\n", hostStat.Hostname)

		// =========== LOAD =============
		loadStat, _ := load.Avg()
		msg.Text += fmt.Sprintf("Load->%.2f\n", loadStat.Load5)

		// =========== MEMORY =============
		memoryStat, _ := mem.VirtualMemory()
		msg.Text += fmt.Sprintf("Ram->%.2f/%.2fGb\n", float64(memoryStat.Used)/1073741824, float64(memoryStat.Total)/1073741824)

		// =========== DISK =============
		var partitions, err = disk.Partitions(false)
		if err != nil {
			log.Panic(err)
		}
		for _, element := range partitions {
			diskStat, err := disk.Usage(element.Mountpoint)
			if strings.HasPrefix(update.Message.Text, "/disk") {
				template := "==== %s ==== \nMount: %s \nTotal: %dGb \nUse: %.2f%% \n"
				msg.Text += fmt.Sprintf(template, element.Device, diskStat.Path, diskStat.Total/1073741824, diskStat.UsedPercent)
			} else {
				if diskStat.Path == "/" {
					template := "Disk->%.2f%%\n"
					msg.Text += fmt.Sprintf(template, diskStat.UsedPercent)
				}
			}
			if err != nil {
				log.Panic(err)
			}
		}

		// =========== TEMPS =============
		tempStat, err := host.SensorsTemperatures()
		tempLast := "INITIAL"
		msg.Text += fmt.Sprintf("Temps->")
		for _, element := range tempStat {
			if element.Temperature > 1 && !strings.HasSuffix(element.SensorKey, "max") && !strings.HasSuffix(element.SensorKey, "min") && !strings.HasSuffix(element.SensorKey, "crit") {
				if tempLast != strings.Split(element.SensorKey, "_")[0] {
					tempLast = strings.Split(element.SensorKey, "_")[0]
					msg.Text += fmt.Sprintf("\n  %s: ", tempLast)
				}
				msg.Text += fmt.Sprintf("%.1f ", element.Temperature)
			}
		}

		// =========== IP =============
		resp, err := http.Get("https://ifconfig.co")
		if err != nil {
			log.Panic(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		msg.Text += fmt.Sprintf("\nIP->%s", body)

		bot.Send(msg)
	}
}
