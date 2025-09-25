package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// 支持的 Alpine 版本列表
	支持的版本列表 := []string{"3.19", "3.20", "3.21", "3.22"}
	// 支持的镜像变体列表
	支持的变体列表 := []string{"default", "cloud"}
	
	// 创建输出目录
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}
	
	// 遍历所有版本和变体进行构建
	for _, 版本 := range 支持的版本列表 {
		for _, 变体 := range 支持的变体列表 {
			log.Printf("正在构建 Alpine %s %s 变体", 版本, 变体)
			
			// 使用 sudo distrobuilder 构建镜像（需要 root 权限）
			构建命令 := exec.Command("sudo", "distrobuilder", "build-lxc", "configs/alpine.yaml", 
				"-o", "image.release="+版本,
				"-o", "image.architecture=x86_64",
				"-o", "image.variant="+变体)
			
			构建命令.Stdout = os.Stdout
			构建命令.Stderr = os.Stderr
			
			log.Printf("执行命令: %s", strings.Join(构建命令.Args, " "))
			
			if err := 构建命令.Run(); err != nil {
				log.Fatalf("构建 Alpine %s %s 失败: %v", 版本, 变体, err)
			}
			
			// 重命名并移动镜像文件
			源文件 := "rootfs.tar.xz"
			目标文件 := fmt.Sprintf("alpine_%s_x86_64_%s.tar.xz", 版本, 变体)
			
			if _, err := os.Stat(源文件); err == nil {
				if err := os.Rename(源文件, filepath.Join("output", 目标文件)); err != nil {
					log.Fatalf("移动镜像文件失败: %v", err)
				}
				log.Printf("✓ 已构建: %s", 目标文件)
			} else {
				log.Printf("⚠ 警告: 未找到文件 %s", 源文件)
			}
		}
	}
	
	log.Println("✅ 所有 Alpine 镜像构建完成")
}
