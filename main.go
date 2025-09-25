package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// 构建任务结构体
type 构建任务 struct {
	版本 string
	变体 string
}

func main() {
	// 支持的 Alpine 版本列表
	支持的版本列表 := []string{"3.19", "3.20", "3.21", "3.22"}
	// 支持的镜像变体列表
	支持的变体列表 := []string{"default", "cloud"}
	
	// 创建输出目录
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}
	
	// 创建所有构建任务
	var 任务列表 []构建任务
	for _, 版本 := range 支持的版本列表 {
		for _, 变体 := range 支持的变体列表 {
			任务列表 = append(任务列表, 构建任务{版本: 版本, 变体: 变体})
		}
	}
	
	// 使用 WaitGroup 等待所有 goroutine 完成
	var wg sync.WaitGroup
	// 创建信号量控制并发数量（避免资源竞争）
	并发限制 := make(chan struct{}, 4) // 同时构建4个镜像
	
	// 错误通道
	错误通道 := make(chan error, len(任务列表))
	
	log.Printf("🚀 开始并发构建 %d 个 Alpine 镜像", len(任务列表))
	
	// 并发执行所有构建任务
	for _, 任务 := range 任务列表 {
		wg.Add(1)
		并发限制 <- struct{}{} // 获取信号量
		
		go func(任务 构建任务) {
			defer wg.Done()
			defer func() { <-并发限制 }() // 释放信号量
			
			log.Printf("📦 开始构建 Alpine %s %s 变体", 任务.版本, 任务.变体)
			
			// 使用 sudo distrobuilder 构建镜像
			构建命令 := exec.Command("sudo", "distrobuilder", "build-lxc", "configs/alpine.yaml", 
				"-o", "image.release="+任务.版本,
				"-o", "image.architecture=x86_64",
				"-o", "image.variant="+任务.变体)
			
			// 捕获命令输出
			var stdout, stderr strings.Builder
			构建命令.Stdout = &stdout
			构建命令.Stderr = &stderr
			
			if err := 构建命令.Run(); err != nil {
				错误信息 := fmt.Sprintf("构建 Alpine %s %s 失败: %v 输出: %s 错误: %s", 
					任务.版本, 任务.变体, err, stdout.String(), stderr.String())
				错误通道 <- fmt.Errorf(错误信息)
				return
			}
			
			// 重命名并移动镜像文件，使用 sudo 确保权限
			源文件 := "rootfs.tar.xz"
			目标文件 := fmt.Sprintf("alpine_%s_x86_64_%s.tar.xz", 任务.版本, 任务.变体)
			
			if _, err := os.Stat(源文件); err == nil {
				// 使用 sudo 移动文件并设置权限
				移动命令 := exec.Command("sudo", "mv", 源文件, filepath.Join("output", 目标文件))
				if err := 移动命令.Run(); err != nil {
					错误通道 <- fmt.Errorf("移动镜像文件 %s 失败: %v", 目标文件, err)
					return
				}
				// 使用 sudo 修改文件权限
				权限命令 := exec.Command("sudo", "chmod", "644", filepath.Join("output", 目标文件))
				if err := 权限命令.Run(); err != nil {
					log.Printf("⚠ 警告: 修改文件 %s 权限失败: %v", 目标文件, err)
				}
				log.Printf("✅ 完成构建: %s", 目标文件)
			} else {
				错误通道 <- fmt.Errorf("未找到构建文件: %s", 源文件)
				return
			}
		}(任务)
	}
	
	// 等待所有任务完成
	wg.Wait()
	close(错误通道)
	
	// 检查是否有错误
	var 错误列表 []string
	for err := range 错误通道 {
		错误列表 = append(错误列表, err.Error())
	}
	
	if len(错误列表) > 0 {
		log.Printf("❌ 构建过程中发生 %d 个错误:", len(错误列表))
		for _, 错误 := range 错误列表 {
			log.Printf("   %s", 错误)
		}
		log.Fatal("构建失败")
	}
	
	log.Printf("🎉 所有 %d 个 Alpine 镜像并发构建完成", len(任务列表))
}
