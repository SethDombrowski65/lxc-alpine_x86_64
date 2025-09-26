package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// 定义命令行参数
	版本 := flag.String("version", "", "Alpine 版本 (3.19|3.20|3.21|3.22)")
	变体 := flag.String("variant", "", "镜像变体 (default|cloud)")
	flag.Parse()

	// 验证参数
	if *版本 == "" || *变体 == "" {
		log.Fatal("❌ 必须指定 --version 和 --variant 参数")
	}

	// 验证版本和变体
	支持的版本 := map[string]bool{"3.19": true, "3.20": true, "3.21": true, "3.22": true}
	支持的变体 := map[string]bool{"default": true, "cloud": true}

	if !支持的版本[*版本] {
		log.Fatalf("❌ 不支持的版本: %s，支持的版本: 3.19|3.20|3.21|3.22", *版本)
	}
	if !支持的变体[*变体] {
		log.Fatalf("❌ 不支持的变体: %s，支持的变体: default|cloud", *变体)
	}

	log.Printf("🚀 开始构建 Alpine %s %s 镜像", *版本, *变体)

	// 创建输出目录
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("❌ 创建输出目录失败: %v", err)
	}

	// 创建工作目录
	工作目录 := fmt.Sprintf("build_%s_%s", *版本, *变体)
	if err := os.MkdirAll(工作目录, 0755); err != nil {
		log.Fatalf("❌ 创建工作目录 %s 失败: %v", 工作目录, err)
	}
	defer func() {
		// 清理临时工作目录
		if err := os.RemoveAll(工作目录); err != nil {
			log.Printf("⚠ 警告: 清理工作目录 %s 失败: %v", 工作目录, err)
		}
	}()

	// 复制配置文件到工作目录
	配置源文件 := filepath.Join("configs", "alpine.yaml")
	配置目标文件 := filepath.Join(工作目录, "alpine.yaml")
	if err := copyFile(配置源文件, 配置目标文件); err != nil {
		log.Fatalf("❌ 复制配置文件失败: %v", err)
	}

	// 切换到工作目录执行构建
	originalDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ 获取当前工作目录失败: %v", err)
	}

	if err := os.Chdir(工作目录); err != nil {
		log.Fatalf("❌ 切换到工作目录 %s 失败: %v", 工作目录, err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			log.Printf("⚠ 警告: 切换回原目录失败: %v", err)
		}
	}()

	// 使用 sudo distrobuilder 构建镜像
	构建命令 := exec.Command("sudo", "distrobuilder", "build-lxc", "alpine.yaml", 
		"-o", "image.release="+*版本,
		"-o", "image.architecture=x86_64",
		"-o", "image.variant="+*变体)

	// 捕获命令输出
	var stdout, stderr strings.Builder
	构建命令.Stdout = &stdout
	构建命令.Stderr = &stderr

	log.Printf("📦 正在构建 Alpine %s %s 镜像...", *版本, *变体)
	if err := 构建命令.Run(); err != nil {
		log.Fatalf("❌ 构建失败: %v\n输出: %s\n错误: %s", err, stdout.String(), stderr.String())
	}

	// 重命名并移动镜像文件，使用 sudo 确保权限
	源文件 := "rootfs.tar.xz"
	目标文件 := fmt.Sprintf("alpine_%s_x86_64_%s.tar.xz", *版本, *变体)

	if _, err := os.Stat(源文件); err != nil {
		log.Fatalf("❌ 未找到构建文件: %s", 源文件)
	}

	// 使用 sudo 移动文件并设置权限
	移动命令 := exec.Command("sudo", "mv", 源文件, filepath.Join(originalDir, "output", 目标文件))
	if err := 移动命令.Run(); err != nil {
		log.Fatalf("❌ 移动镜像文件 %s 失败: %v", 目标文件, err)
	}

	// 使用 sudo 修改文件权限
	权限命令 := exec.Command("sudo", "chmod", "644", filepath.Join(originalDir, "output", 目标文件))
	if err := 权限命令.Run(); err != nil {
		log.Printf("⚠ 警告: 修改文件 %s 权限失败: %v", 目标文件, err)
	}

	log.Printf("✅ 完成构建: %s", 目标文件)
	log.Printf("🎉 Alpine %s %s 镜像构建成功", *版本, *变体)
}

// 复制文件函数
func copyFile(源文件, 目标文件 string) error {
	输入, err := os.ReadFile(源文件)
	if err != nil {
		return err
	}
	return os.WriteFile(目标文件, 输入, 0644)
}
