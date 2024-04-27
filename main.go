package main

import (
	"compress/gzip"
	"dockerpull/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

func main() {
	port, _ := getArgs()
	startHttp(port)
}

func getArgs() (int64, string) {
	args := os.Args
	l := len(args)

	portstr := "8888"
	authstr := ""
	if l >= 2 {
		// http port given
		portstr = args[1]
	}

	if l >= 3 {
		authstr = args[2]
	}

	port, err := strconv.ParseInt(portstr, 10, 64)
	if err != nil {
		log.Fatalln("cannot parse port:", portstr)
	}

	return port, authstr
}

func startHttp(port int64) {
	r := mux.NewRouter().StrictSlash(false)
	r.HandleFunc("/", serveIndexPage).Methods("GET") // 使用新的函数来服务主页

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong")
	})

	r.HandleFunc("/events", sseHandler)

	r.HandleFunc("/pull", pull)
	r.HandleFunc("/download", downloadImage)

	n := negroni.New()
	n.UseHandler(r)

	log.Println("listening http on", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), n))
}

func pull(w http.ResponseWriter, r *http.Request) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	//从请求的query参数：image 获取值
	if r.URL.Query().Get("image") == "" {
		http.Error(w, "image is empty", http.StatusBadRequest)
		return
	}
	image := r.URL.Query().Get("image")

	sh := "docker pull " + string(image)

	cmd := exec.Command("sh", "-c", sh)

	utils.HandleCommandOutput(cmd, w, flusher)

}

// serveIndexPage 用于服务静态的index.html文件
func serveIndexPage(w http.ResponseWriter, r *http.Request) {
	// 假设index.html位于项目的根目录下
	http.ServeFile(w, r, "./index.html")
}

func compressTarToGz(src string) (string, error) {
	file, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipFilePath := src + ".gz"
	gzipFile, err := os.Create(gzipFilePath)
	if err != nil {
		return "", err
	}
	defer gzipFile.Close()

	writer := gzip.NewWriter(gzipFile)
	defer writer.Close()

	_, err = io.Copy(writer, file)
	return gzipFilePath, err
}

func downloadImage(w http.ResponseWriter, r *http.Request) {

	imageName := r.URL.Query().Get("image")
	exportName := r.URL.Query().Get("exportName")
	if imageName == "" {
		http.Error(w, "Image name is required", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("exportName") == "" {
		http.Error(w, "exportName is empty", http.StatusBadRequest)
		return
	}

	// 执行docker save命令并捕获输出到临时文件
	currentDir, _ := os.Getwd()
	tempDir := filepath.Join(currentDir, "temp")

	//检查 tempDir 是否存在
	CreateDir(tempDir)
	tempTarPath := filepath.Join(tempDir, exportName+".tar")
	saveCmd := exec.Command("docker", "save", "-o", tempTarPath, imageName)
	if err := saveCmd.Run(); err != nil {
		log.Printf("Error saving Docker image: %v", err)
		http.Error(w, "Failed to save Docker image", http.StatusInternalServerError)
		return
	}
	gzipFilePath, err := compressTarToGz(tempTarPath)
	if err != nil {
		log.Printf("Error compress file: %v", err)
		http.Error(w, "Failed to compress gzip file", http.StatusInternalServerError)
		return
	}

	defer os.Remove(tempTarPath)

	// 设置响应头准备下载
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", exportName))
	w.Header().Set("Content-Type", "application/gzip")
	// 读取并发送.gz文件
	gzippedFile, err := os.Open(gzipFilePath)
	if err != nil {
		log.Printf("Error opening gzip file for sending: %v", err)
		http.Error(w, "Failed to open gzip file", http.StatusInternalServerError)
		return
	}
	defer gzippedFile.Close()

	_, err = io.Copy(w, gzippedFile)
	if err != nil {
		log.Printf("Error sending file: %v", err)
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
	}
	//下载完成后清理文件
	os.Remove(gzipFilePath)
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 设置必要的响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 循环发送消息
	for {
		message := fmt.Sprintf("Data: %s\n\n", time.Now().Format(time.RFC3339))
		_, err := w.Write([]byte(message))
		if err != nil {
			return // 发生错误时中断循环
		}
		flusher.Flush()

		time.Sleep(2 * time.Second) // 模拟延迟，实际应用中可能依据具体需求调整
	}
}

func CreateDir(dir string) {
	_, err := os.Stat(dir)
	if err == nil {
		// fmt.Println(dir, "目录已存在")
		return
	}
	// 创建文件夹
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Println(dir, "目录创建出错:", err)
	} else {
		fmt.Println(dir, "目录创建成功")
	}
}
