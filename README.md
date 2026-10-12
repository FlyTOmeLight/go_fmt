<!-- Keep these links. Translations will automatically update with the README. -->
[Deutsch](https://zdoc.app/de/fsgo/go_fmt) | 
[English](https://zdoc.app/en/fsgo/go_fmt) | 
[Español](https://zdoc.app/es/fsgo/go_fmt) | 
[français](https://zdoc.app/fr/fsgo/go_fmt) | 
[日本語](https://zdoc.app/ja/fsgo/go_fmt) | 
[한국어](https://zdoc.app/ko/fsgo/go_fmt) | 
[Português](https://zdoc.app/pt/fsgo/go_fmt) | 
[Русский](https://zdoc.app/ru/fsgo/go_fmt) | 
[中文](https://zdoc.app/zh/fsgo/go_fmt)

# Go 代码格式化

## 1 功能说明
* 自动补全、清理 import，默认按 `标准库`、`第三方库`、`项目自身库` 分组；支持 Go 工作区
* 格式化单行注释：若为 `//注释内容`，调整为 `//{空格}注释内容`
* 默认只对 git 项目库里有修改的进行格式化
* 支持将多行的 copyright 注释修改为单行格式(默认不调整)
* 简化代码、重写修正(将废弃的方法替换为推荐的新方法)
* struct 赋值表达式，自动补齐 key
* 严格排版，补充空行、移除多余的空行
* 默认按 120 列目标宽度拆分长行，可用 `-max-len` 调整
* 提供全局 push 前检查，阻止未格式化的提交推送

对于 import 部分：
> 1.可使用`-mi`参数来控制是否将多段import合并为一段（默认合并）。
> 2.对于注释的import path,会按照其实际路径参与分组和排序。   
> 3.对于非末行的注释的位置会和其下面紧挨的import path绑定在一起。  
> 4.末行的注释则会放入import的头部。  
> 5.import path 不要使用相对路径(如`./` 和 `../`)。

会忽略当前目录以及子目录下的 `testdata` 和 `vendor` 目录。  
若需要可进入其目录里执行该命令。  

<details><summary><i>Example 1：补齐 struct key</i></summary>

``` diff
- u2 := User{"hello", 12}
+ u2 := User{Name: "hello", Age: 12}
```
</details>

<details><summary><i>Example 2：注释格式化</i></summary>

```diff
- //User 注释内容
+ // User 注释内容
type User struct{
```
</details>

<details><summary><i>Example 3：简化、重写代码</i></summary>

1. 简化循环逻辑：
```diff
- s[a:len(s)]
+ s[a:]

- for x, _ = range v {...}
+ for x = range v {...}

- for _ = range v {...}
+ for range v {...}

- for {
+ for ok{
-   if !ok {
-     break
-   }
   // do something
 }
```

2. 简化判断逻辑：
```diff
- if b == true {
+ if b { 

- if b == false {
+ if !b {

- if b != true {
+ if !b {

- if b != false {
+ if b {

- for b == true {
+ for b {

- _ = 1 == index
+ _ = index == 1

- _ = 1 < index
+ _ = index > 1

 func ok() bool {
- 	if a > b {
- 		return true
- 	}else{
- 		return false
+ 	return a > b
} 

func ok() bool {
- 	if a > b {
- 		return true
- 	return false
+ 	return a > b
} 

- if val!=nil && len(val)!=0 {
+ if len(val)!=0 {
   // do something
}

```

3. 使用 `strings.Contains` 替换 `strings.Count` 和 `strings.Index`
```diff
- strings.Count(s, "a") == 0
+ !strings.Contains(s, "a")

- strings.Count(s, "a") > 0
+ strings.Contains(s, "a")

- strings.Count(s, "a") != 0
+ strings.Contains(s, "a")
```
`bytes.Count` 具有和 `strings.Count` 一样的规则。

```diff
- strings.Index(s, "a") == -1
+ !strings.Contains(s, "a")

- strings.Index(s, "a") != -1
+ strings.Contains(s, "a")
```
`bytes.Index` 具有和 `strings.Index` 一样的规则。

4. 字符串的比较：

使用 `bytes.Equal` 替换 `bytes.Compare`：
```diff
- bytes.Compare(s,a) == 0
+ bytes.Equal(s, a)

- bytes.Compare(s,a) != 0
+ !bytes.Equal(s, a)
```

使用 `==` 替换 `strings.Compare`：
```diff
- strings.Compare("abc","a") == 0
+ "abc" == "a"

- strings.Compare("abc","a") != 0
+ "abc" != "a"
```

5. 递增 1、递减 1:
```diff
- i += 1
+ i++

- i -= 1
+ i--
```

6. time.Since 和 time.Until
`time.Since`  替换 `time.Now().Sub`:
```diff
- time.Now().Sub( t1 )
+ time.Since( t1 )
```

`time.Until`  替换 `t.Sub( time.Now() )`:
```diff
- t1.Sub( time.Now() )
+ time.Until( t1 )
```

7. channel:
```diff
- _ = <-chan
+ <-done
```

8. map:
```diff
- x, _ := someMap["key"]
+ x := someMap["key"]
```

9. fmt:
```diff
- fmt.Errorf("hello")
+ errors.New("hello")

- fmt.Printf("abc")
+ fmt.Print("abc")

- log.Printf("abc")
+ log.Print("abc")

- _ = errors.New(fmt.Sprintf("hello"))
+ _ = errors.New("hello")

- _ = errors.New(fmt.Sprintf("hello %s", "world"))
+ _ = fmt.Errorf("hello %s", "world")

- bf.Write([]byte(fmt.Sprintf("hello %d", 1)))
+ fmt.Fprintf(bf,"hello %d",1)

- fmt.Sprintf("%d",123)  // 性能  1
+ strconv.Atoi(123)      // 性能  3

- fmt.Sprintf("%v",123)
+ strconv.Atoi(123)   

- fmt.Sprintf("%d",int32Num)
+ strconv.FormatInt(int64(int32Num), 10)

- fmt.Sprintf("%d",uint32Num)
+ strconv.FormatUint(uint64(uint32Num), 10)
```

10. raw string :
```diff
- regexp.Compile("\\A(\\w+) profile: total \\d+\\n\\z")
+ regexp.Compile(`\A(\w+) profile: total \d+\n\z`)
```

11. sort :
```diff
- sort.Sort(sort.StringSlice(x))
+ sort.Strings(x)
```

12. os.IsExist 等替换为使用 errors.Is
```diff
- os.IsExist(err)
+ errors.Is(err, fs.ErrExist)

- os.IsNotExist(err)
+ errors.Is(err, fs.ErrNotExist)

- os.IsPermission(err)
+ errors.Is(err, fs.ErrPermission)
```
添加于： 2025-09-02 v0.7.0


</details>

<details><summary><i>Example 4：基于表达式规则，重写代码</i></summary>
使用 `-rr=false` 可以使用默认内置规则不生效。

1. 替换废弃的 `ioutil` 的函数调用：

```diff
import (
-	"io/ioutil"
+	"io
)

- buf, err := ioutil.ReadAll(f)
+ buf, err := io.ReadAll(f)
```
</details>

<details><summary><i>Example 5：移除多余的空行</i></summary>

1. 移除 struct 内部前后多余的空行：
```diff
type userfn91 struct{
-				
	name string
-				
}
```

2. 移除 func 内部前后多余的空行：
``` diff
fn1() {
-				
	println("hello")
-				
}

```

3. 空 func 变为一行：
```diff
- fn1() {
- }
+ fn1() {}
```
</details>

<details><summary><i>Example 6：补充空行</i></summary>
在适当的位置添加空行可以增加代码的可读性。

1. struct 有文档的字段前后添加换行：

```diff
type User1 struct {
-				
	// on Name
	Name string
+				
	// on Age
	Age int
+				
	Grade int
	Class int

	Address string // 前面有空行，会保持
}
```

2. interface 有文档的方法前后添加换行：

```diff
type Group1 interface {
-				
	// Register 注册延迟函数
	Register(fn func())
+				
	Add()
+				
	// on Delete
	Delete()
+				
	Fn1()

	Fn2() // 前面有空行，会保持
-				
}
```

3. 多个定义之间添加空行:

```diff
type (
	User1 struct {
		name string
	}
+				
	User1 struct {
-				
		name string
	}
)
```

4. 全局的，不同类型定义之间添加空行：

```diff
var a="hello"
var b="world" // after b
+				
const c01="say"
+ 				
var a0 = "a0"
```

</details>


<details><summary><i>Example 7：Array / Slice 格式化</i></summary>

```diff
- var _ = []int{
-	1, 2, 
-	3, 4, 5}
// 当代码是如上这种 3 行格式的时候（这 3 行内不能有注释），会格式化为下面这样。
// 上面第一行是 2 个元素，所以按照每行 2 个元素格式化对齐。

+	var _ = []int{
+	 1, 2,
+	 3, 4,
+	 5,
+	 }
```
</details>

## 2 安装/更新

需要 Go 1.26 或更高版本。从本仓库安装当前源码：

```bash
git clone https://github.com/FlyTOmeLight/go_fmt.git
cd go_fmt
go install ./cmd/gorgeous
```

命令安装到 `GOBIN`，未设置时安装到 `$(go env GOPATH)/bin`。将此目录加入 `PATH`。更新时，在仓库中执行 `git pull --ff-only`，再执行 `go install ./cmd/gorgeous`。

也可以从 [Releases](https://github.com/FlyTOmeLight/go_fmt/releases) 下载系统对应的二进制，重命名为 `gorgeous`，放入 `PATH` 中的目录。macOS 和 Linux 需要执行 `chmod +x gorgeous`。升级 Go 后，建议重新编译安装。

本仓库保留原有模块路径 `github.com/fsgo/go_fmt`。原上游的 `go install ...@latest` 不会安装本仓库新增的默认格式化流程。

## 3 使用

### 3.0 help
> gorgeous -help

```bash
usage: gorgeous [flags] [path ...]
  -d	display diffs instead of rewriting files
  -df string
    	display diffs format, support: text, json (default "text")
  -format-only
      skip code rewrites and type loading
  -max-len int
      target line length (default 120, 0 disables) (default 120)
  -srcdir string
      real source filename for stdin/editor input
  -e	enable extra rules (default true)
  -ig string
    	import groups sort rule,
    	stc: Go Standard package, Third Party package, Current package
    	sct: Go Standard package, Current package, Third Party package
    	 (default "stc")
  -local string
    	current package path, will put imports beginning with this string as 3rd-party packages.
    	by default, it will got from 'go.mod' file.
    	 (default "auto")
  -mi
    	merge imports into one section.
    	with env 'GORGEOUS_MI=false' to set default value as false (default true)
  -r value
    	rewrite rule (e.g., 'a[b:len(a)] -> a[b:]')
    	or a file path for rewrite rules (like -rr)

  -rr
    	rewrite with build in rules:
    	a[b:len(a)] -> a[b:]
    	interface{} -> any                    // go1.18
    	io/#ioutil.NopCloser -> io.NopCloser  // go1.16
    	io/#ioutil.ReadAll   -> io.ReadAll    // go1.16
    	io/#ioutil.ReadFile  -> os.ReadFile   // go1.16
    	io/#ioutil.TempFile  -> os.CreateTemp // go1.16
    	io/#ioutil.TempDir   -> os.MkdirTemp  // go1.16
    	io/#ioutil.WriteFile -> os.WriteFile  // go1.16
    	io/#ioutil.Discard   -> io.Discard    // go1.16

    	with env 'GORGEOUS_RR=false' to set default value as false
    	 (default true)
  -s	simplify code (default true)
  -slcr
    	multiline copyright to single-line
    	with env 'GORGEOUS_SLCR=true' to set default value as true (default true)
  -trace
    	show trace messages
  -w	write result to (source) file instead of stdout (default true)
```
### 3.1 格式化 `git` 项目里有修改的`.go`文件
```bash
$ gorgeous
```

### 3.2 对当前目录所有 `.go` 文件格式化
```bash
$ gorgeous ./...
```

### 3.3 对指定 `.go` 文件格式化
```bash
$ gorgeous abc.go
```

### 3.4 从 STDIN 读取代码并输出到 STDOUT
```bash
$ cat code.go|gorgeous stdin
```

### 3.5 默认格式化流程

直接运行 `gorgeous` 会格式化 Git 中有变更的 Go 文件并写回。`gorgeous ./...` 处理整个目录树，`gorgeous -d` 仅显示差异。

默认流程将各项目的能力融合进 gorgeous：先执行已有的代码简化与内置重写，再补全缺失的 import、删除未使用的 import，清理多余空行并统一排版，按 120 列目标宽度拆分长行，最后按“标准库、第三方、当前模块”分组。无需选择工具或另行安装格式化命令。

排版复用 gofumpt，导入补删复用 goimports，长行处理复用 golines，默认导入分组复用 GCI。gorgeous 负责统一规则顺序、模块识别、类型约束和输出。版本固定在 `go.mod` 中。

| 参数 | 作用 |
| --- | --- |
| `-format-only` | 跳过 struct 补 key、代码简化和表达式重写，不加载包类型信息；仍执行默认排版、导入修复和长行处理。 |
| `-max-len N` | 调整目标行宽，默认 `120`，`0` 关闭长行处理。长字符串、部分单行函数等不能保证拆分。 |
| `-ig stc\|sct` | 调整导入分组顺序，沿用现有参数。 |
| `-srcdir path/to/file.go` | 为 stdin 提供真实文件名，用于识别模块、工作区和补全 import。 |

```bash
gorgeous -d ./...          # 预览整个仓库的默认格式化结果
gorgeous -max-len 100 ./... # 用 100 列目标行宽格式化并写回
cat code.go | gorgeous -format-only -srcdir /path/to/project/code.go stdin
```

默认分组按源文件位置识别 `go.work`，工作区内模块归入当前模块组；支持 `GOWORK` 和 `GOWORK=off`。显式 `-local`、API 的 `ImportGroupFn` / `ThirdModules` 和 `-mi=false` 继续使用其指定的导入策略。补删 import 仍会整理导入声明，`-mi=false` 不保证保留原始声明的所有空行和分隔。

`-format-only` 不能与 `-r` 一起使用。省略它时，原有 `-s`、`-rr` 和 struct 补 key 行为仍然有效。格式化不会自动执行 `go mod tidy`；缺少依赖时请显式准备依赖，或使用 `-format-only`。类型信息不完整时跳过依赖类型的修正。`-d` 不写源文件、`go.mod` 或 `go.sum`；检测到差异以非零状态退出。

### 3.6 官方 Go 代码现代化

`gorgeous fix` 将后续参数交给本机 Go 工具链的 `go fix`，使用其类型分析、语言版本约束和逐规则开关。此入口独立于格式化参数；先现代化，再按需要运行格式化。

```bash
gorgeous fix -diff ./...  # 预览，不写文件
gorgeous fix -any ./...   # 仅运行 interface{} -> any 规则
gorgeous fix ./...        # 应用当前 Go 工具链支持的规则
```

需要 Go 1.26 或更高版本。可用规则随本机 Go 版本变化；使用 `go tool fix help` 查看。`gorgeous fix` 保留 `go fix` 的输出与退出码。

### 3.7 在 Go 中使用

```go
opt := gofmtapi.NewOptions()
opt.FormatOnly = true
opt.MaxLineLength = 100
_, formatted, _, err := gofmtapi.NewFormatter().Format("/path/to/project/code.go", source, opt)
```

传入的 `source` 优先于磁盘内容。返回的 `formatted` 是格式化后的字节；是否实际处理该文件由第三个返回值表示。生成文件仍遵循原有跳过规则。第三方许可见 [THIRD_PARTY_LICENSES.txt](THIRD_PARTY_LICENSES.txt)。

## 4 Git push 前自动检查

安装 `gorgeous` 后，在本仓库执行：

```bash
sh scripts/install-git-hooks.sh
```

这会将 `core.hooksPath` 全局设置为 `${XDG_CONFIG_HOME:-$HOME/.config}/git/hooks`。所有使用该全局设置的仓库在 push 前检查实际要推送的提交快照。Go 文件使用 gorgeous 默认规则检查；非 Go 仓库、删除远端分支会跳过格式检查。多个分支或标签指向同一提交时，只检查一次。

检查不会改动工作区。格式不合格时 push 会停止，先整理并提交改动，再重试：

```bash
gorgeous ./...
git add path/to/changed.go
git commit -m "format Go code"
git push
```

安装脚本保留各仓库原有 `.git/hooks/` 的行为，包括原有 `pre-push` 的参数与输入。已有其他全局 hooks 配置或同名自定义脚本时，安装脚本会停止，并提示将 `scripts/pre-push` 加入现有 hook 链。仓库单独设置的 `core.hooksPath` 会覆盖全局设置；这类仓库需要将检查加入自己的 hook 链。非标准位置的可执行文件可用 `GORGEOUS_BIN` 指定。

检查提交快照需要 Git、tar 和 Go 工具链。依赖和本地 `replace` 路径也必须能在提交快照中解析；格式化不会执行 `go mod tidy`。

## 5 GitHub Actions
```yml
- name: Set up Go
  uses: actions/setup-go@v4
  with:
    go-version: 1.27

- name: gorgeous style check 
  # gorgeous 添加 “-d” 参数后，只检查代码风格，不重写文件，会在终端输出需要格式化的文件信息，退出码为非 0
  run: go install ./cmd/gorgeous && gorgeous -d ./...
```

## 6 Visual Studio Code

### 6.1 As goformat
1. Install as goformat:
```bash
go install ./cmd/goformat
```
2. 配置的 `Go: Format Tool`，设置为 "goformat"：
```
  "go.formatTool": "goformat"
```

### 6.2 Run on Save
1. 先安装插件 [Run on Save](https://marketplace.visualstudio.com/items?itemName=pucelle.run-on-save) <img src="https://github.com/pucelle/vscode-run-on-save/raw/master/images/logo.png" width="20px"/>

2. 配置插件，在保存文件的时候执行格式化命令：
```json
  "runOnSave.commands": [
    {
        "match": "\\.go$",
        "command": "cd ${fileDirname} && gorgeous -rr ${fileBasename}",
        "runIn":"terminal"
    }
 ]
```
3. 配置的 `Go: Format Tool`，设置为 "default"：
```
  "go.formatTool": "default"
```
