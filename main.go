package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	// 调用API获取时间
	dateTime := GetBilibiliTime()
	if dateTime != "" {
		UpdateSystemDate(dateTime)
	} else {
		glog.Info("未能获取到有效的时间")
	}
}

func GetBilibiliTime() string {
	// 发送HTTP请求
	resp, err := http.Get("https://api.bilibili.com/x/report/click/now")
	if err != nil {
		glog.Error("请求失败:", err)
		return ""
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
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
	biliTime := time.Unix(timeResponse.Data.Now, 0)
	return biliTime.Format("2006-01-02 15:04:05")
}

func UpdateSystemDate(dateTime string) bool {
	// 获取当前系统时间
	oldTime := time.Now()
	fmt.Println("原先的系统时间:", oldTime.Format("2006-01-02 15:04:05"))

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
		fmt.Println("更改后的系统时间:", newTime.Format("2006-01-02 15:04:05"))

		// // 计算时间偏差
		// timeDifference := newTime.Sub(oldTime)
		// fmt.Println("时间偏差:", timeDifference)
		return true
	}

	return false
}
