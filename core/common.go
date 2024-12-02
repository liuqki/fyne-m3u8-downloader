package core

import (
	"os"
	"path/filepath"
)

// 单个下载任务
type Downloader struct {
	VideoUrl    string
	VideoTitle  string // 视频名-一般是集数
	SavePath    string
	DownPercent float32 // 下载进度（单位1）
	JumpHeadSec uint32  // 跳过片头的长度
	JumpTailSec uint32  // 跳过片尾的长度
	isBatch     bool    // 是否是批量下载
}

// 视频片段
type Segment struct {
	Duration float64 // 片段长度，秒
	Title    string  // 标题
	Url      string  // 地址

	filePath string // 本地保存地址
}

type M3u8 struct {
	Version    int8       // m3u8文件版本号，一般为3
	Duration   float64    // 片段总长度，秒
	BandWidth  string     // 最大码率
	Resolution string     // 分辨率
	IsEnd      bool       // 是否到了索引文件的结尾
	Segments   []*Segment // 视频片段切片
}

// 获取当前可执行文件所在目录
func GetCurrentDir() (string, error) {
	return filepath.Abs(filepath.Dir(os.Args[0]))
}
