package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gen2brain/beeep"
)

// ConfigFileName 配置文件名
var ConfigFileName = "config.json"

// Config 配置
type Config struct {
	Address string `json:"address"`
	APIKey  string `json:"apikey"`
	Title   string `json:"title"`
	Since   int    `json:"since"`
}

// Event 事件
type Event struct {
	ID       int       `json:"id"`
	GlobalID int       `json:"globalID"`
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
	Data     struct {
		Action string `json:"action"`
		Error  string `json:"error"`
		Folder string `json:"folder"`
		Item   string `json:"item"`
		Type   string `json:"type"`
	} `json:"data"`
}

// flagConfig 命令行参数配置
var flagConfig Config

var defaultAddress = "http://127.0.0.1:8384"
var defaultAPIKey = ""
var defaultTitle = "提醒"

func init() {
	// 命令行参数解析
	flag.StringVar(&flagConfig.Address, "address", defaultAddress, "syncthing web gui address")
	flag.StringVar(&flagConfig.APIKey, "apikey", defaultAPIKey, "syncthing api key")
	flag.StringVar(&flagConfig.Title, "title", defaultTitle, "notify title")

	flag.Parse()
}

func main() {
	config := LoadConfig()
	st := NewSyncTray()
	st.Run()

	url := fmt.Sprintf("%s/rest/events", config.Address)
	since := config.Since
	for {
		var events []Event
		// 构造请求查看有没有新文件下载完成
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			fmt.Println(err)
			continue
		}
		req.Header.Set("X-API-Key", config.APIKey)
		sinceStr := strconv.Itoa(since)
		q := req.URL.Query()
		q.Add("events", "ItemFinished")
		q.Add("since", sinceStr)
		req.URL.RawQuery = q.Encode()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("error request", err)
			continue
		}
		// 返回结果处理
		err = json.NewDecoder(resp.Body).Decode(&events)
		resp.Body.Close()
		if err != nil {
			fmt.Println("error parse json", err)
			continue
		}
		// 桌面提醒
		for _, event := range events {
			msg := fmt.Sprintf("%s 有变动", event.Data.Item)
			go st.FlashTray(event.Data.Item)
			err := MsgNotify(config.Title, msg)
			if err != nil {
				fmt.Println("error notify", err)
				continue
			}
		}
		// 更新最新起始位与配置文件
		if len(events) > 0 {
			since = events[len(events)-1].ID
			config.Since = since
			UpdateConfig(config)
		}
	}
}

// IsExist 检查文件是否存在
func IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// LoadConfig 加载配置文件
func LoadConfig() (config Config) {
	isExists := IsExist(ConfigFileName)
	if isExists {
		configData, err := ioutil.ReadFile(ConfigFileName)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(configData, &config)
		if err != nil {
			panic(err)
		}
	}
	if flagConfig.Address != defaultAddress || !isExists {
		config.Address = flagConfig.Address
	}
	if flagConfig.APIKey != defaultAPIKey || !isExists {
		config.APIKey = flagConfig.APIKey
	}
	if flagConfig.Title != defaultTitle || !isExists {
		config.Title = flagConfig.Title
	}
	return
}

// UpdateConfig 更新配置文件
func UpdateConfig(config Config) {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println("error json.Marshal", err)
		return
	}
	err = ioutil.WriteFile("config.json", data, 0644)
	if err != nil {
		fmt.Println("error write config", err)
	}
}

// MsgNotify 消息弹框提醒
func MsgNotify(title string, msg string) error {
	err := beeep.Notify(title, msg, "")
	return err
}
