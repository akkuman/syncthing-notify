// +build linux darwin

package main

// SyncTray 托盘图标
type SyncTray struct {
}

// NewSyncTray 初始化托盘
func NewSyncTray() (syncTray SyncTray) {
	return
}

// FlashTray 托盘闪烁与消息
func (st *SyncTray) FlashTray(msg string) {
	return
}

// Run 托盘启动
func (st *SyncTray) Run() {
	return
}
