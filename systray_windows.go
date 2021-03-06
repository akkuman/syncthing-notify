// +build windows,!linux,!freebsd,!netbsd,!openbsd,!darwin,!js

package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/tadvi/systray"
)

// SyncTray 托盘图标
type SyncTray struct {
	Tray     *systray.Systray
	CommIcon systray.HICON
	TranIcon systray.HICON
	MsgChan  chan string
	IsFlash  bool
}

var (
	// CommIconBase64 图标base64数据
	CommIconBase64 = "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"
	// TranIconBase64 透明图标base64数据
	TranIconBase64 = "AAABAAEAEBAAAAEAIACDAAAAFgAAAIlQTkcNChoKAAAADUlIRFIAAAAQAAAAEAEDAAAAJT1tIgAAAAFzUkdCAdnJLH8AAAAJcEhZcwAAAnYAAAJ2Adpg408AAAADUExURQAAAKd6PdoAAAABdFJOUwBA5thmAAAADElEQVR4nGNgIA0AAAAwAAEWiZrRAAAAAElFTkSuQmCC"
	// CommIconPath 图标临时文件路径
	CommIconPath = filepath.Join(os.TempDir(), "systray_temp_icon_comm")
	// TranIconPath 透明图标临时文件路径
	TranIconPath = filepath.Join(os.TempDir(), "systray_temp_icon_tran")
)

func init() {
	// 写入temp ico
	err := writeBase64File(CommIconBase64, CommIconPath)
	if err != nil {
		panic(err)
	}
	err = writeBase64File(TranIconBase64, TranIconPath)
	if err != nil {
		panic(err)
	}
}

// NewSyncTray 初始化托盘
func NewSyncTray() (syncTray SyncTray) {
	st, err := systray.New()
	if err != nil {
		panic(err)
	}
	commIcon, err := loadIconFile(CommIconPath)
	tranIcon, err := loadIconFile(TranIconPath)
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
	st.Tray.AppendMenu("Exit", func() {
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

// WriteBase64File 写文件
func writeBase64File(base64Date string, tmpFilePath string) (err error) {
	iconBytes, err := base64.StdEncoding.DecodeString(base64Date)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(tmpFilePath, iconBytes, 0644)
	return
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
