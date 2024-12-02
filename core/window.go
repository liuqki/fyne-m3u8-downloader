package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var logCh chan string // 用户传输日志的管道

// 填充窗口内容
func FillWindow(win fyne.Window) {
	videoUrlEntry := widget.NewMultiLineEntry()
	videoUrlEntry.SetPlaceHolder("支持批量下载，每行一个链接")

	savePathEntry := widget.NewEntry()
	savePathEntry.SetPlaceHolder("设置视频保存地址")

	sePathBtn := widget.NewButton("选择文件夹", func() {
		// 也可以使用 NewFolderOpen() 并结合 show()来展示文件夹选择器
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if lu != nil {
				savePathEntry.SetText(lu.Path())
			}
		}, win)
	})
	savePathCon := container.New(layout.NewVBoxLayout(), savePathEntry, container.New(layout.NewHBoxLayout(), sePathBtn))

	jumpHeadEntry := widget.NewEntry()
	jumpHeadEntry.SetPlaceHolder("跳过片头的长度（近似值）")

	jumpTailEntry := widget.NewEntry()
	jumpTailEntry.SetPlaceHolder("跳过片尾的长度（近似值）")

	proBar := widget.NewProgressBar()
	proBar.Hidden = true

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "视频链接", Widget: videoUrlEntry},
			{Text: "存储路径", Widget: savePathCon},
			{Text: "跳过片头x秒", Widget: jumpHeadEntry},
			{Text: "跳过片尾x秒", Widget: jumpTailEntry},
		},
		OnSubmit: func() {
			urlReg := regexp.MustCompile(`((.+)\$)?(https?:\/\/.+\.m3u8)$`)
			var urlSlice []string
			if runtime.GOOS == "windows" {
				// win平台上，复制粘贴和手打的换行的换行标识不一致
				urlSlice = strings.Split(videoUrlEntry.Text, "\r\n")
				if len(urlSlice) == 0 {
					urlSlice = strings.Split(videoUrlEntry.Text, "\n")
				}
			} else {
				urlSlice = strings.Split(videoUrlEntry.Text, "\n")
			}

			var titleList []string
			var urlList []string
			for k, v := range urlSlice {
				v = strings.Trim(v, " ")
				if v == "" {
					continue
				}
				matchRes := urlReg.FindStringSubmatch(v)

				if len(matchRes) == 0 {
					dialog.ShowError(fmt.Errorf("第%d行输入的.m3u8视频地址有误", k+1), win)
					return
				}

				titleList = append(titleList, matchRes[2]) // 集数
				urlList = append(urlList, matchRes[3])     // 地址
			}

			isBatch := false // 本次是否批量下载
			if len(urlList) == 0 {
				dialog.ShowError(errors.New("请输入有效的.m3u8视频地址"), win)
				return
			} else if len(urlList) > 1 {
				isBatch = true
			}

			savePath := strings.Trim(savePathEntry.Text, " /\\")
			if savePath == "" {
				dialog.ShowError(errors.New("请选择存储路径"), win)
				return
			}
			savePath += string(os.PathSeparator) // 拼接上目录分隔符

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

			// 开始循环去下载视频了
			for k, v := range urlList {
				down := &Downloader{
					VideoUrl:    v,
					VideoTitle:  titleList[k],
					SavePath:    savePath,
					JumpHeadSec: uint32(headNum),
					JumpTailSec: uint32(tailNum),
					isBatch:     isBatch,
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
			}

		},
		SubmitText: "下载",
	}
	formBox := container.New(layout.NewVBoxLayout(), form, proBar)

	// we can also append items
	// form.Append("Text", textArea)

	win.SetContent(container.New(layout.NewGridLayout(1), formBox, initLogsArea()))
}

// 构建日志区域--可竖向滚动
func initLogsArea() fyne.CanvasObject {
	bindStr := binding.NewString()
	intro := widget.NewLabelWithData(bindStr)
	intro.Wrapping = fyne.TextWrapWord
	intro.Show()

	// 用一个协程来监听日志管道，并写入日志区域
	logCh = make(chan string, 10)
	go func(b binding.String) {
		for {
			newLog := <-logCh
			str, _ := b.Get()
			b.Set(str + "\n" + newLog)
		}
	}(bindStr)

	return container.NewVScroll(container.NewVBox(intro))
}
