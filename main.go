package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/tadvi/systray"
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

// SyncTray 托盘图标
type SyncTray struct {
	Tray     *systray.Systray
	CommIcon systray.HICON
	TranIcon systray.HICON
	MsgChan  chan string
	IsFlash  bool
}

// flagConfig 命令行参数配置
var flagConfig Config

var defaultAddress = "http://127.0.0.1:8384"
var defaultAPIKey = ""
var defaultTitle = "提醒"

func init() {
	flag.StringVar(&flagConfig.Address, "address", defaultAddress, "syncthing web gui address")
	flag.StringVar(&flagConfig.APIKey, "apikey", defaultAPIKey, "syncthing api key")
	flag.StringVar(&flagConfig.Title, "title", defaultTitle, "notify title")

	flag.Parse()
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

// loadIconFile 加载ico
func loadIconFile(file string) (systray.HICON, error) {
	path, err := filepath.Abs(file)
	if err != nil {
		return 0, err
	}
	icon, err := systray.NewIconFromFile(path)
	if err != nil {
		return 0, err
	}
	return systray.HICON(icon), nil
}

// NewSyncTray 初始化托盘
func NewSyncTray() (syncTray SyncTray) {
	st, err := systray.New()
	if err != nil {
		panic(err)
	}
	commIcon, err := loadIconFile("ico.ico")
	tranIcon, err := loadIconFile("trans.ico")
	if err != nil {
		panic(err)
	}
	st.SetIcon(commIcon)
	st.SetTooltip("SyncNotify")
	err = st.SetVisible(true)
	if err != nil {
		panic(err)
	}
	syncTray.Tray = st
	syncTray.CommIcon = commIcon
	syncTray.TranIcon = tranIcon
	syncTray.MsgChan = make(chan string)

	return
}

// FlashTray 托盘闪烁与消息
func (st *SyncTray) FlashTray(msg string) {
	// 将文件变动消息加入菜单
	st.Tray.AppendMenu(msg, func() {
		var index = 0
		for i := range st.Tray.Menu {
			index++
			if st.Tray.Menu[i].Label == msg {
				break
			}
		}
		st.Tray.Menu = append(st.Tray.Menu[:index-1], st.Tray.Menu[index:]...)
	})
	st.MsgChan <- msg
}

// Run 托盘启动
func (st *SyncTray) Run() {
	st.Tray.OnClick(func() {
		st.IsFlash = false
	})
	// 加入退出菜单
	st.AppendMenu("Exit", func() {
		os.Exit(0)
	})
	go func() {
		count := 0
		st.IsFlash = false
		for {
			select {
			case <-st.MsgChan:
				st.IsFlash = true
				st.Tray.SetTooltip("点击查看变动")
			default:
				if !st.IsFlash {
					st.Tray.SetIcon(st.CommIcon)
					continue
				}
				if count%2 == 0 {
					st.Tray.SetIcon(st.CommIcon)
				} else {
					st.Tray.SetIcon(st.TranIcon)
				}
				count++
				time.Sleep(300 * time.Millisecond)
			}
		}
	}()
	go st.Tray.Run()
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
