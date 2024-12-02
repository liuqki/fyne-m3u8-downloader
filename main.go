package main

import (
	"fmt"
	"hello/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("v0.3.0")
	ii, _ := fyne.LoadResourceFromPath("./Icon.png")
	a.SetIcon(ii)
	win := a.NewWindow("Go-GUI-m3u8 v0.3.0") // 创建一个窗口
	win.Resize(fyne.NewSize(800, 600))       // 设置窗口的大小

	// uri := "https://v6.tlkqc.com/wjv6/202309/09/mB8fcA4D031/video/index.m3u8"
	// subUrl := "https://m3u.nikanba.live/share/0e9c7d6985b4436d25a19e33351f4c68.m3u8"
	// subUrl = "https://m3u8.heimuertv.com/play/51f2ec1cdfda48f6978bf81b8b384b45.m3u8" // 子ts带后缀的
	// subUrl = "https://c1.7bbffvip.com/video/fanrenxiuxianchuan/第123集/index.m3u8" // 子ts不带http前缀的
	// https://v11.tlkqc.com/wjv11/202406/24/PCY7CLew6k83/video/index.m3u8
	// https://v11.tlkqc.com/wjv11/202406/24/PCY7CLew6k83/video/1000k_720/hls/index.m3u8

	// 捕捉错误
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	core.FillWindow(win)

	win.Show() // 展示这个窗口
	a.Run()    // 让项目跑起来
}
