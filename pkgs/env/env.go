// Package env 提供构建期注入的元信息，不依赖任何业务代码，可独立复制到其他
// Go 项目复用。
//
// 变量在编译期通过 -ldflags "-X" 覆盖，未注入时使用默认值：
//
//	go build -trimpath -ldflags "\
//	  -X <module>/pkgs/env.AppName=myapp \
//	  -X <module>/pkgs/env.Version=v1.2.3 \
//	  -X <module>/pkgs/env.GitCommit=a1b2c3d \
//	  -X <module>/pkgs/env.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
//
// 注意：-X 只能作用于包级 string 变量，且初始值必须是常量表达式（不能是函数调用）。
package env

import (
	"fmt"
	"runtime"
)

var (
	// AppName 应用名称，可选；未注入时版本信息中不显示。
	AppName = ""

	// Version 版本号，如 v1.2.3。
	Version = "dev"

	// GitCommit 构建所用的提交哈希，建议使用短 SHA。
	GitCommit = "unknown"

	// BuildDate 构建时间，建议使用 UTC + RFC3339。
	BuildDate = "unknown"
)

// Info 构建信息快照。
type Info struct {
	AppName   string `json:"appName"`
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// GetInfo 返回当前二进制的构建信息；运行期字段由 runtime 提供，无需注入。
func GetInfo() Info {
	return Info{
		AppName:   AppName,
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回单行构建信息，便于日志与命令行展示。
func (i Info) String() string {
	detail := fmt.Sprintf("(%s, build %s, %s, %s)", i.GitCommit, i.BuildDate, i.GoVersion, i.Platform)
	if i.AppName != "" {
		return fmt.Sprintf("%s %s %s", i.AppName, i.Version, detail)
	}
	return fmt.Sprintf("%s %s", i.Version, detail)
}

// PrintVersion 输出构建信息，供 -version 场景调用。
func PrintVersion() {
	fmt.Println(GetInfo())
}
