package main

import (
	"errors"
	"fmt"
	"hello/tool"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	ii, _ := fyne.LoadResourceFromPath("./Icon.png")
	a.SetIcon(ii)
	win := a.NewWindow("Go-GUI-m3u8 V0.2.0") // 创建一个窗口
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

	videoUrlEntry := widget.NewEntry()
	videoUrlEntry.SetPlaceHolder("https://")
	videoUrlEntry.SetText("https://v11.tlkqc.com/wjv11/202406/24/PCY7CLew6k83/video/index.m3u8")

	savePathEntry := widget.NewEntry()
	savePathEntry.SetPlaceHolder("设置视频保存地址")
	savePathEntry.SetText("F:/vTmp/")

	jumpHeadEntry := widget.NewEntry()
	jumpHeadEntry.SetPlaceHolder("跳过片头的长度（近似值）")

	jumpTailEntry := widget.NewEntry()
	jumpTailEntry.SetPlaceHolder("跳过片尾的长度（近似值）")

	proBar := widget.NewProgressBar()
	proBar.Hidden = true

	bindStr := binding.NewString()

	var logCh = make(chan string, 10)
	go func(ss binding.String) {
		for {
			newLog := <-logCh
			str, _ := ss.Get()
			ss.Set(str + "\n" + newLog)
		}
	}(bindStr)
	// textArea := widget.NewMultiLineEntry()

	intro := widget.NewLabelWithData(bindStr)
	intro.Wrapping = fyne.TextWrapWord
	intro.Show()
	logsArea := container.NewVScroll(container.NewVBox(intro))

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "视频链接", Widget: videoUrlEntry},
			{Text: "存储地址", Widget: savePathEntry},
			{Text: "跳过片头x秒", Widget: jumpHeadEntry},
			{Text: "跳过片尾x秒", Widget: jumpTailEntry},
		},
		OnSubmit: func() {
			urlReg := regexp.MustCompile(`^https?://.+\.m3u8$`)
			if isUrl := urlReg.Match([]byte(videoUrlEntry.Text)); !isUrl {
				dialog.ShowError(errors.New("请输入有效的.m3u8视频地址"), win)
				return
			}
			var headNum, tailNum int64
			var err error

			if strings.Trim(jumpHeadEntry.Text, " ") != "" {
				headNum, err = strconv.ParseInt(jumpHeadEntry.Text, 10, 32)
				if err != nil || headNum < 0 {
					dialog.ShowError(errors.New("《跳过片头》请输入正整数"), win)
					return
				}
			}

			if strings.Trim(jumpTailEntry.Text, " ") != "" {
				tailNum, err = strconv.ParseInt(jumpTailEntry.Text, 10, 32)
				if err != nil || tailNum < 0 {
					dialog.ShowError(errors.New("《跳过片尾》请输入正整数"), win)
					return
				}
			}

			down := &tool.Downloader{
				VideoUrl:    videoUrlEntry.Text,
				SavePath:    savePathEntry.Text,
				JumpHeadSec: uint32(headNum),
				JumpTailSec: uint32(tailNum),
			}
			proBar.SetValue(0)
			proBar.Hidden = false
			// w.Close()
			err = down.DownVideo(proBar, logCh)
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				proBar.SetValue(1)
			}
		},
		SubmitText: "下载",
	}
	formBox := container.New(layout.NewVBoxLayout(), form, proBar)

	// we can also append items
	// form.Append("Text", textArea)

	win.SetContent(container.New(layout.NewGridLayout(1), formBox, logsArea))

	win.Show() // 展示这个窗口
	a.Run()    // 让项目跑起来
}
