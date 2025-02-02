package vhost

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	frpLog "github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"
	"sync/atomic"
)

var (
	ServiceUnavailablePagePath string
	ServiceUnavailable         = `
<html lang="zh">
	<head>
		<meta charset="UTF-8">
		<title>凌云FRP-无法找到您所请求的网站</title>
		<link rel="stylesheet" href="https://frp.lybyte.cn/assets/frp/style.css">
		<link rel="shortcut icon" href="https://frp.lybyte.cn/favicon.ico" type="image/x-icon">
		<script src="https://frp.lybyte.cn/assets/frp/prefixfree.min.js"></script>
		<script src="https://frp.lybyte.cn/assets/frp/jquery.min.js"></script>
		<script src="https://frp.lybyte.cn/assets/frp/script.js"></script>
		<style>
			body {
				margin: 0;
				overflow: hidden;
			}
	
			#canvasContainer {
				position: absolute;
				width: 100%;
				height: 100%;
				z-index: -1;
			}
	
			canvas {
				width: 100%;
				height: 100%;
			}
		</style>
	</head>
	<body class="grid" style="overflow:hidden;">
		<div id="canvasContainer">
			<canvas id="myCanvas"></canvas>
		</div>
		<main class="grid" style="--n: 3; --k: 0">
			<article class="grid" id="a0" style="--i: 0">
				<h3 class="c--ini fade">凌云FRP-错误警告</h3>
				<p class="c--ini fade">无法找到您所请求的网站,请确定凌云FRP服务已正常启动并检查内网端口是否正确。Unable to find the website you requested.
					Please ensure that the LingYunFRP service has started properly</p><a class="nav c--ini fade"
					href="#a1">Next</a>
				<section class="grid c--fin" role="img" aria-label="错误页面随机图片"
					style="--img: url(https://uapis.cn/api/imgapi/bq/youshou.php); --m: 8">
					<div class="slice" aria-hidden="true" style="--j: 0"></div>
					<div class="slice" aria-hidden="true" style="--j: 1"></div>
					<div class="slice" aria-hidden="true" style="--j: 2"></div>
					<div class="slice" aria-hidden="true" style="--j: 3"></div>
					<div class="slice" aria-hidden="true" style="--j: 4"></div>
					<div class="slice" aria-hidden="true" style="--j: 5"></div>
					<div class="slice" aria-hidden="true" style="--j: 6"></div>
					<div class="slice" aria-hidden="true" style="--j: 7"></div>
				</section><a class="det grid c--fin fade" href="https://frp.lybyte.cn">访问凌云FRP</a>
			</article>
			<article class="grid" id="a1" style="--i: 1">
				<h3 class="c--ini fade">可能的原因</h3>
				<p class="c--ini fade">1：您尚未启动映射。2：错误的本地端口。3：错误的外网端口。4：映射协议错误。5：nginx或apache或iis错误</p><a
					class="nav c--ini fade" href="https://frp.lybyte.cn">Next</a>
				<section class="grid c--fin" role="img" aria-label="错误页面随机图片"
					style="--img: url(https://uapis.cn/api/imgapi/bq/maomao.php); --m: 8">
					<div class="slice" aria-hidden="true" style="--j: 0"></div>
					<div class="slice" aria-hidden="true" style="--j: 1"></div>
					<div class="slice" aria-hidden="true" style="--j: 2"></div>
					<div class="slice" aria-hidden="true" style="--j: 3"></div>
					<div class="slice" aria-hidden="true" style="--j: 4"></div>
					<div class="slice" aria-hidden="true" style="--j: 5"></div>
					<div class="slice" aria-hidden="true" style="--j: 6"></div>
					<div class="slice" aria-hidden="true" style="--j: 7"></div>
				</section><a class="det grid c--fin fade" href="https://frp.lybyte.cn">访问凌云FRP</a>
			</article>
		</main>
	</body>
	<script src="https://frp.lybyte.cn/assets/frp/frp.js"></script>
	<script>
		var canvas = document.getElementById("myCanvas");
		var ctx = canvas.getContext("2d");
	</script>
	</html>
`
)
var (
	serviceUnavailableCount    uint64
	maxServiceUnavailableCount uint64
)

// 添加一个函数来设置 maxServiceUnavailableCount
func SetMaxServiceUnavailableCount(count uint64) {
	maxServiceUnavailableCount = count
	resetServiceUnavailableCount()
}

// 重置 serviceUnavailableCount 的定时器
func resetServiceUnavailableCount() {
	go func() {
		for {
			time.Sleep(1 * time.Minute) // 每分钟重置一次
			atomic.StoreUint64(&serviceUnavailableCount, 0)
		}
	}()
}

func getServiceUnavailablePageContent() []byte {
	atomic.AddUint64(&serviceUnavailableCount, 1)
	if serviceUnavailableCount > maxServiceUnavailableCount {
		return []byte{} // 返回空字节切片
	} else {
		if ServiceUnavailablePagePath != "" {
			buf, err := ioutil.ReadFile(ServiceUnavailablePagePath)
			if err != nil {
				frpLog.Warn("read custom 503 page error: %v", err)
				return []byte(ServiceUnavailable)
			}
			return buf
		} else {
			return []byte(ServiceUnavailable)
		}
	}
}

func notFoundResponse() *http.Response {
	if serviceUnavailableCount > maxServiceUnavailableCount {
		return &http.Response{
			Status:     "403 Forbidden",
			StatusCode: 403,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
			ProtoMinor: 0,
			Header:     make(http.Header),
			Body:       http.NoBody, // 直接使用 http.NoBody 表示不返回任何内容
		}
	} else {
		header := make(http.Header)
		header.Set("server", "frp/"+version.Full()+"-sakurapanel")
		header.Set("Content-Type", "text/html")

		res := &http.Response{
			Status:     "Service Unavailable",
			StatusCode: 503,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
			ProtoMinor: 0,
			Header:     header,
			Body:       ioutil.NopCloser(bytes.NewReader(getServiceUnavailablePageContent())),
		}
		return res
	}
}

func noAuthResponse() *http.Response {
	header := make(map[string][]string)
	header["WWW-Authenticate"] = []string{`Basic realm="Restricted"`}
	res := &http.Response{
		Status:     "401 Not authorized",
		StatusCode: 401,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}

// 示例处理请求的函数
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// 检查是否应该返回403 Forbidden
	if serviceUnavailableCount > maxServiceUnavailableCount {
		resp := notFoundResponse()
		resp.Write(w)
		return
	}

	// 其他处理逻辑
	pageContent := getServiceUnavailablePageContent()
	w.Header().Set("Content-Type", "text/html")
	w.Write(pageContent)
}
