package tool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2/widget"
)

var completeNum int
var totalNum int
var wg sync.WaitGroup

// 下载.m3u8
func (dd Downloader) DownVideo(bar *widget.ProgressBar, logCh chan string) error {
	var goNum = runtime.NumCPU()     // 获取cpu的逻辑核心数作为开启下载协程的数量
	idxCh := make(chan int, goNum*2) // 定义存放下载索引的管道
	completeNum = 0
	start := time.Now()
	logCh <- fmt.Sprintf("%s 开始处理：%s", start.Format("15:04:05"), dd.VideoUrl)

	m3u8 := &M3u8{}
	err := parseUrl(dd.VideoUrl, m3u8)
	if err != nil {
		logCh <- "url解析失败"
		return err
	}

	if len(m3u8.Segments) == 0 {
		logCh <- "没有匹配到下载链接"
		return nil
	}

	err = dd.jump(m3u8, logCh) // 跳过片头片尾
	if err != nil {
		logCh <- fmt.Sprintf("%v", err)
		return err
	}

	_, err = os.Stat(dd.SavePath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(dd.SavePath, 0666)
			if err != nil {
				logCh <- fmt.Sprintf("创建下载文件夹失败，错误：%v", err)
				return err
			}
		} else {
			logCh <- fmt.Sprintf("创建下载文件夹失败，错误：%v", err)
			return err
		}
	}

	fileList := make([]string, 100)
	subNameReg := regexp.MustCompile(`([\w-]+)\.(ts|jpeg)`) // 匹配出文件名的正则
	totalNum = len(m3u8.Segments)
	logCh <- fmt.Sprintf("共需下载%d个片段，开始下载...", totalNum)

	go func() {
		for k, v := range m3u8.Segments {
			var res [][]byte
			var fileName string // 文件名

			res = subNameReg.FindSubmatch([]byte(v.Url))
			if res != nil {
				fileName = fmt.Sprintf("%v.%v", string(res[1]), string(res[2]))
			} else {
				fileName = fmt.Sprintf("%v%v.mp4", time.Now().Unix(), rand.Intn(1000))
			}
			fullName := dd.SavePath + fileName    // 拼接保存文件的全文件名
			fileList = append(fileList, fullName) // 记录全文件名
			v.filePath = fullName
			idxCh <- k
		}
		close(idxCh)
	}()

	for i := 0; i < goNum; i++ {
		wg.Add(1)
		go download(m3u8, bar, logCh, idxCh)
	}

	wg.Wait()
	bar.SetValue(1) // 设置进度条为100%
	logCh <- "下载已完成，开始合并..."
	newFile := fmt.Sprintf("%v%v.mp4", dd.SavePath, time.Now().Unix()) // 最终保存文件的名字
	combine(newFile, fileList, logCh)
	logCh <- fmt.Sprintf("合并已完成，视频保存路径：%v，共耗时：%v\n", newFile, time.Since(start))
	return nil
}

// 解析m3u8文件
func parseUrl(videoUrl string, m3 *M3u8) error {
	var reader *bufio.Reader
	var lNum int     // m3u8文件行号
	var seg *Segment // 临时存储片段信息的指针

	// 寻找下载链接的前半部分
	endReg := regexp.MustCompile(`\w+\.m3u8$`)
	preUrl := endReg.ReplaceAllString(videoUrl, "")
	uriReg := regexp.MustCompile(`https?://.*`)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", videoUrl, nil)
	resp, e1 := client.Do(req)
	if e1 != nil {
		return e1
	}
	defer resp.Body.Close()
	reader = bufio.NewReader(resp.Body)

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return errors.New("读取出错")
			}
		}

		lineStr := string(line)

		// 文件第一行是m3u8文件的标识符
		if lNum == 0 {
			if lineStr != "#EXTM3U" {
				return errors.New("不是合法的m3u8文件")
			}
			lNum++
			continue
		}

		switch {
		case lineStr == "":
			continue
		case strings.HasPrefix(lineStr, "#EXT-X-VERSION:"):
			fmt.Sscanf(lineStr, "#EXT-X-VERSION:%d", &m3.Version)
		case strings.HasPrefix(lineStr, "#EXT-X-STREAM-INF:"):
			m3.Resolution = lineStr // 临时记录主文件信息
		case strings.HasPrefix(lineStr, "#EXTINF"):
			// 处理片段时长及标题
			tt := regexp.MustCompile(`#EXTINF:([\d.]+),(.*)?`)
			subMatch := tt.FindSubmatch(line)
			if seg == nil {
				seg = new(Segment)
			}
			if subMatch[1] != nil {
				seg.Duration, _ = strconv.ParseFloat(string(subMatch[1]), 32)
			}
			if subMatch[2] != nil {
				seg.Title = string(subMatch[2])
			}
		case !strings.HasPrefix(lineStr, "#"):
			// 记录子片段下载地址
			if seg == nil {
				// 这种情况一般只出现在m3u8嵌套的情况
				seg = new(Segment)
			}

			if uriReg.Match(line) {
				seg.Url = lineStr
			} else {
				seg.Url = preUrl + lineStr
			}

			m3.Segments = append(m3.Segments, seg)
			seg = nil
		case lineStr == "#EXT-X-ENDLIST":
			m3.IsEnd = true
		}

		lNum++
	}

	// 如果没有找到m3u8的结尾字符，一般是多嵌套了一层，再往下找一层
	if !m3.IsEnd && len(m3.Segments) > 0 {
		lastPart := m3.Segments[len(m3.Segments)-1]
		m3.Segments = nil
		return parseUrl(lastPart.Url, m3)
	}

	return nil
}

// 处理跳过片头片尾
func (dd Downloader) jump(m3 *M3u8, logCh chan string) error {
	// 跳过片头
	if dd.JumpHeadSec > 0 {
		var headIdx = -1 // 跳过片头的索引
		var headTotal float64
		for k, v := range m3.Segments {
			if dd.JumpHeadSec > uint32(headTotal) && dd.JumpHeadSec >= uint32(headTotal+v.Duration) {
				headTotal += v.Duration
			} else {
				headIdx = k
				break
			}
		}
		if headIdx == -1 {
			logCh <- "跳过片头的时间超长了"
			return errors.New("跳过片头的时间超长了")
		}
		logCh <- fmt.Sprintf("跳过片头%.2f秒", headTotal)
		m3.Segments = m3.Segments[headIdx:]
	}

	// 跳过片尾
	if dd.JumpTailSec > 0 {
		var tailIdx = -1 // 跳过片尾的索引
		var tailTotal float64
		var lens = len(m3.Segments)
		for i := lens - 1; i >= 0; i-- {
			if dd.JumpTailSec > uint32(tailTotal) && dd.JumpTailSec >= uint32(tailTotal+m3.Segments[i].Duration) {
				tailTotal += m3.Segments[i].Duration
			} else {
				tailIdx = i
				break
			}
		}
		if tailIdx == -1 {
			logCh <- "跳过片头的时间超长了"
			return errors.New("跳过片头的时间超长了")
		}
		if tailIdx < lens {
			tailIdx++
			logCh <- fmt.Sprintf("跳过片尾%.2f秒", tailTotal)
			m3.Segments = m3.Segments[:tailIdx]
		}
	}

	return nil
}

// 单个协程执行单个文件下载
func download(m3 *M3u8, bar *widget.ProgressBar, logCh chan string, idxCh chan int) error {
	for idx := range idxCh {
		seg := m3.Segments[idx]
		client := &http.Client{}
		req, _ := http.NewRequest("GET", seg.Url, nil)

		resp, err := client.Do(req)
		if err != nil {
			logCh <- fmt.Sprintf("http发送失败！链接：%s。错误：%v", seg.Url, err)
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// 打开指定文件，并设置文件打开的模式及权限
		file, err := os.OpenFile(seg.filePath, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			logCh <- fmt.Sprintf("文件创建失败！文件名：%s。错误：%v", seg.filePath, err)
			return err
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		writer.Write(body)
		writer.Flush() // 将缓存中的数据写入文件。

		completeNum++
		bar.SetValue(float64(completeNum) / float64(totalNum))
	}

	wg.Done()
	return nil
}

// 全部文件合并
func combine(fileName string, subFileList []string, logCh chan string) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logCh <- fmt.Sprintf("合并时，文件创建失败！文件名：%s。错误：%v", fileName, err)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for k, v := range subFileList {
		if v == "" {
			continue
		}
		tt, err := os.ReadFile(v)
		if err != nil {
			logCh <- fmt.Sprintf("合并时，片段文件读取失败！第%d个片段，文件名：%s。错误：%v", k+1, v, err)
			return err
		}
		_, err = writer.Write(tt)
		if err != nil {
			logCh <- fmt.Sprintf("合并第%d个片段失败！文件名：%s。错误：%v", k+1, v, err)
			return err
		}
		os.Remove(v)
	}

	writer.Flush() // 将缓存中的数据写入文件。
	return nil
}
