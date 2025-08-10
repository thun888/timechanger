package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/os/gproc"
	"github.com/gogf/gf/text/gstr"
)

type BilibiliTimeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		Now int64 `json:"now"`
	} `json:"data"`
}

func main() {
	// 解析命令行参数
	jumpTimePtr := flag.Float64("jumptime", 0, "跳转时间(小时为单位)")
	flag.Parse()

	// 如果指定了jumptime参数
	if *jumpTimePtr != 0 {
		// 获取当前时间并添加偏移量
		currentTime := time.Now()
		hoursToAdd := time.Duration(*jumpTimePtr * float64(time.Hour))
		newTime := currentTime.Add(hoursToAdd)

		// 格式化时间并更新系统时间
		dateTime := newTime.Format("2006-01-02 15:04:05")
		if UpdateSystemDate(dateTime) {
			glog.Info("已成功将系统时间调整 ", *jumpTimePtr, " 小时")
		} else {
			glog.Error("调整系统时间失败")
		}
		return
	}

	// 调用API获取时间，无限重试直到成功
	dateTime := GetBilibiliTimeWithRetry()
	if dateTime != "" {
		// 解析从B站获取的时间字符串
		biliTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateTime, time.Local)
		if err != nil {
			glog.Error("解析B站时间失败:", err)
			return
		}

		// 获取当前系统时间
		localTime := time.Now()

		// 计算时间差异，显示绝对值和方向
		timeDiff := localTime.Sub(biliTime)
		diffDirection := "快"
		if timeDiff < 0 {
			timeDiff = -timeDiff
			diffDirection = "慢"
		}

		// 更新系统时间
		if UpdateSystemDate(dateTime) {
			glog.Info(fmt.Sprintf("系统时间已校准，校准前系统时间%s了 %v", diffDirection, timeDiff))
		} else {
			glog.Error("调整系统时间失败")
		}
	} else {
		glog.Info("未能获取到有效的时间")
	}
}

// GetBilibiliTimeWithRetry 带无限重试功能的哔哩哔哩时间获取
func GetBilibiliTimeWithRetry() string {
	retryCount := 0
	for {
		dateTime := GetBilibiliTime()
		if dateTime != "" {
			if retryCount > 0 {
				glog.Info(fmt.Sprintf("在尝试%d次后成功获取时间", retryCount+1))
			}
			return dateTime
		}

		retryCount++
		glog.Info(fmt.Sprintf("第%d次请求失败，15秒后重试...", retryCount))
		time.Sleep(15 * time.Second) // 重试前等待15秒
	}
}

func GetBilibiliTime() string {
	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送HTTP请求
	resp, err := client.Get("http://api.bilibili.com/x/report/click/now")
	if err != nil {
		glog.Error("请求失败:", err)
		return ""
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		glog.Error("读取响应内容失败:", err)
		return ""
	}

	// 解析JSON数据
	var timeResponse BilibiliTimeResponse
	if err := json.Unmarshal(body, &timeResponse); err != nil {
		glog.Error("解析JSON失败:", err)
		return ""
	}

	// 确认返回的数据有效
	if timeResponse.Code != 0 {
		glog.Error("API返回错误:", timeResponse.Message)
		return ""
	}

	// 将时间戳转换为本地时间格式
	biliTime := time.Unix(timeResponse.Data.Now, 0).Local()
	return biliTime.Format("2006-01-02 15:04:05")
}

func UpdateSystemDate(dateTime string) bool {
	// 获取当前系统时间
	oldTime := time.Now()
	glog.Info("原先的系统时间:", oldTime.Format("2006-01-02 15:04:05"))

	// 解析目标时间 - 使用本地时区
	targetTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateTime, time.Local)
	if err != nil {
		glog.Error("解析目标时间失败:", err)
		return false
	}
	glog.Info("目标时间:", dateTime)

	system := runtime.GOOS
	var success bool

	switch system {
	case "windows":
		{
			_, err1 := gproc.ShellExec(`date  ` + gstr.Split(dateTime, " ")[0])
			_, err2 := gproc.ShellExec(`time  ` + gstr.Split(dateTime, " ")[1])
			if err1 != nil && err2 != nil {
				glog.Info("更新系统时间错误:请用管理员身份启动程序!")
				success = false
			} else {
				success = true
			}
		}
	case "linux", "darwin":
		{
			_, err1 := gproc.ShellExec(`date -s  "` + dateTime + `"`)
			if err1 != nil {
				glog.Info("更新系统时间错误:", err1.Error())
				success = false
			} else {
				success = true
			}
		}
	default:
		return false
	}

	if success {
		// 获取更改后的系统时间
		newTime := time.Now()
		glog.Info("更改后的系统时间:", newTime.Format("2006-01-02 15:04:05"))

		// 计算与目标时间的误差，显示绝对值和方向
		timeDifference := newTime.Sub(targetTime)
		diffDirection := "快"
		if timeDifference < 0 {
			timeDifference = -timeDifference
			diffDirection = "慢"
		}
		glog.Info(fmt.Sprintf("校准后系统时间%s了 %v", diffDirection, timeDifference))
		return true
	}

	return false
}
