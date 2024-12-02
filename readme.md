# fyne-m3u8-downloader

fyne-m3u8-downloader是一款基于Go语言fyne工具包开发的跨平台m3u8下载器  
主要用途是学习

## 待完成
- [ ] 多视频支持合并成一个合集。
- [ ] 优化错误处理：普通错误跳过；致命错误停止运行
- [ ] 跳过片头片尾结合图像识别功能：针对片头片尾时间不固定的视频，选择片头、片尾画面截图，使用图像对比来找到更精确的进度


## v0.3.0
2024.12.2

新特性：
- [x] 支持下载批量m3u8视频链接。
- [x] 可以通过选择文件夹的方式，指定保存下载文件的路径。


## v0.2.0
2024.10.28

新特性：
- [x] 下载进度条在初始打开软件时不展示，开启下载任务后才展示。
- [x] 增加下载进度日志展示区域，该区域需要支持使用滚动条来容纳更多内容。
- [x] 下载内容视频可以选择跳过开头xx秒和结尾xx秒。（进阶采用图像识别对比来自行根据指定图片来跳过片头片尾）
