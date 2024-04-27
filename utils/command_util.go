package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
)

// 实时输出命令的结果
func HandleCommandOutput(cmd *exec.Cmd, w http.ResponseWriter, flusher http.Flusher) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Println("开始执行命令:", cmd.String())

	// 创建管道
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	// 实时输出命令的结果
	go func() {
		defer pw.Close()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := w.Write([]byte("data: " + line + "\n\n"))
			if err != nil {
				log.Println("无法写入SSE数据:", err)
				return
			}
			flusher.Flush()
			fmt.Println(line)
		}
	}()

	err := cmd.Start()
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
