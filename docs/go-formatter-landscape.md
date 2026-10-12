# Go 格式化工具与升级机会

调研日期：2026-10-12。仅采用官方文档、项目源码和发行说明。编辑器与 CI 集成可证明采用情况；GitHub stars 只能说明关注度，不能证明好评或正确性。

下文的仓库评估记录升级前状态。后续实施结果见文末“融合结果”。

## 主流工具的职责

| 工具 | 适用范围 | 维护与采用证据 |
| --- | --- | --- |
| `gofmt` / `go fmt` | Go 默认格式；`gofmt -s` 做有限语法简化，`-r` 按表达式模式改写。 | Go 工具链内置；提供 stdin、文件、目录和 diff 模式。[官方命令说明](https://pkg.go.dev/cmd/gofmt) |
| `goimports` | 补全缺失 import、删除未使用 import，并输出 gofmt 风格。 | Go 官方 `x/tools` 项目；适合保存时执行。[官方文档](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) |
| `gofumpt` | 比 gofmt 更严格的空行、布局等规则；输出仍兼容 gofmt。 | `gopls` 原生支持；最新发行页列出 `v0.12.0`，基于 Go 1.27，要求 Go 1.26+。[编辑器设置](https://go.dev/gopls/settings#formatting)、[发行说明](https://github.com/mvdan/gofumpt/releases/tag/v0.12.0) |
| `gci` | 确定性 import 分组；可区分标准库、多个前缀、别名和本地模块。 | GolangCI-Lint 集成；`v0.14.0` 增加 `go.work` 本地模块识别。[README](https://github.com/daixiang0/gci)、[发行说明](https://github.com/daixiang0/gci/releases/tag/v0.14.0) |
| `golines` | 按行长拆分代码。 | 原 `segmentio/golines` 于 2025-12-19 归档，官方指向 `golangci/golines`；维护分支发行页列出 `v0.16.0`。[归档说明](https://github.com/segmentio/golines#maintenance--archiving)、[维护分支发行页](https://github.com/golangci/golines/releases) |

GolangCI-Lint 同时支持上述五类 formatter，这是实际生态集成证据，不能据此推断它们同样适合本项目。[支持列表](https://golangci-lint.run/docs/formatters/)

## 对设计有价值的区别

`gofumpt` 冻结 Go printer 等实现，固定工具版本即可固定格式行为。升级说明还专门修复了 import 注释错位，说明注释归属应作为回归测试重点。[设计说明](https://github.com/mvdan/gofumpt#updating-with-goformat-and-cmdgofmt)、[注释修复](https://github.com/mvdan/gofumpt/releases/tag/v0.12.0)

`goimports -format-only` 将纯排版与 import 修复分开；`-srcdir` 让编辑器缓冲区按真实目录解析 import。其候选包搜索有名称启发式，不应当作任意语义重写的类型证明。[CLI 源码](https://github.com/golang/tools/blob/master/cmd/goimports/goimports.go)、[解析源码](https://github.com/golang/tools/blob/master/internal/imports/fix.go)

Go 1.26 已重写 `go fix`，通过 `go/analysis` 提供数十种现代化改写。支持 `-diff`、独立规则开关和 Go 版本门控，也跳过生成文件。它按当前构建配置分析，覆盖多平台需分别运行。项目自有简化规则应先与其比较，再决定保留范围。[官方介绍](https://go.dev/blog/gofix)、[Go 1.26 发行说明](https://go.dev/doc/go1.26#tools)

## 可借鉴的工程做法

建议提供职责清楚的模式：排版、import 整理、类型相关改写。固定 formatter 与 CI 版本，并检查重复运行不再产生变化。`gci` 的分组配置适合有明确组织规则的项目；只有确有行长需求才启用 `golines`。[配置示例](https://golangci-lint.run/docs/formatters/configuration/)

性能方面，官方 `gofmt` 并发处理文件，按输入字节数限制并发开销，同时按确定顺序输出。可借鉴这一目标，但本次未进行跨工具性能实测，不能给出速度排名。[实现](https://github.com/golang/go/blob/master/src/cmd/gofmt/gofmt.go)

## 当前仓库评估

建议继续维护 `gorgeous`，重点保留 Git 变更文件选择、项目 import 分组、注释和空行规则等已有价值。先修复正确性与检查模式，再明确纯格式和语义改写的边界。暂不把所有外部 formatter 接入一个流水线，也不直接更换默认排版引擎；这会改变现有输出契约，需要先用现有 fixtures 评估。

以下结论来自当前源码及本机隔离实验。P0 表示第一批必须处理，P1 表示下一批优化。

| 优先级 | 发现与影响 | 具体升级方向 |
| --- | --- | --- |
| P0 | `xtest.CheckFile` 发现输出与 `.want` 不同，只写 `.got`，未报告测试失败。故意使用错误 `.want` 的临时测试仍 PASS。 | 在差异分支使用 `t.Errorf`，检查期望文件读取错误；保留 `.got` 供排查。先修测试保障，再处理重写。[位置](../internal/xtest/file.go#L61) |
| P0 | CLI 不论 `-d`、是否启用重写，均执行 `go mod tidy`。隔离模块中，`-d sample.go` 删除了未使用的 `require`。 | 从格式化路径移除自动 tidy。依赖准备由用户显式执行；检查模式不能修改源码、`go.mod` 或 `go.sum`。[调用](../internal/gofmt/go_fmt.go#L54)、[实现](../internal/xpasser/mod.go#L21) |
| P0 | 语义重写依赖标识符名称。局部 `strings` 对象的 `Replace` 被改为不存在的 `ReplaceAll`；自定义 `Clock.Sub(time.Time) int` 被改为 `time.Until(c)`，造成编译错误。 | 用 `go/types` 确认真实包、函数和接收者类型。类型信息缺失或出错时跳过依赖类型的规则。补包别名、名称遮蔽、自定义接收者的反例。[名称判断](../internal/simplify/common.go#L90)、[Replace](../internal/simplify/custom_callexpr.go#L89)、[Until](../internal/simplify/custom_callexpr.go#L151) |
| P0 | race 构建在 50 个文件的 `-d ./...` 检查中报告数据竞争，涉及 `loadConfig` 与 `execCallBack`。后者并发追加共享 diff 切片。 | 在启动 worker 前完成配置初始化；worker 返回独立结果，由单一收集阶段按输入顺序汇总、输出。验证无数据竞争且 JSON/text diff 顺序固定。[配置](../internal/localmodule/config.go#L25)、[收集](../internal/gofmt/go_fmt.go#L127) |
| P0 | 本机 Go 1.27.2 全量测试失败：`TestFormatter_Execute/app2` 报告 export data version 5 超过解码支持 version 4。当前 `x/tools` 为 v0.49.0。`Load` 还只检查顶层 error，未汇总 `Package.Errors`。 | 核实并升级支持目标工具链的 `x/tools` 及配套依赖；Go 1.26、1.27 分别验证类型加载。显式处理包错误，避免错误类型信息参与重写。尚未验证哪个新版本能解决，不能把依赖升级视为已完成修复。[依赖](../go.mod)、[加载](../internal/xpasser/prog.go#L111)、[官方错误处理示例](https://pkg.go.dev/golang.org/x/tools/go/packages#example-package) |
| P1 | `fmtSprintfInt` 未检查参数数量就访问 `Args[0]`、`Args[1]`。`fmt.Sprintf("%d")` 隔离样例导致进程 panic。 | 先检查参数数量；单文件错误应返回带文件与规则上下文的 error。此类不完整代码常见于编辑器缓冲区。[位置](../internal/simplify/custom_callexpr.go#L472) |
| P1 | 即使只格式化一个文件，执行流程仍以 `./...` 加载当前目录包的语法、类型和测试。每条 rewrite 还重复解析输出。 | 建立不加载包类型的纯排版路径；仅对实际需要类型的规则，按目标文件所属包加载并在一次执行内复用。先测单文件和全仓耗时，再决定是否合并重解析，避免破坏 token/comment 位置。[流程](../internal/gofmt/go_fmt.go#L56)、[Load](../internal/xpasser/prog.go#L91)、[Rewrites](../internal/simplify/api.go#L145) |
| P1 | CI 声明了 OS matrix，但 `runs-on` 固定为 Ubuntu，实际没有跨平台运行；静态检查工具使用 `@latest`。 | 将 runner 绑定 matrix，按 GitHub 当时可用 runner 标签维护；固定 staticcheck 版本，增加多文件 diff 的 race 检查。[配置](../.github/workflows/go.yml#L12) |

### 重写误判的最小例子

合法输入如下。`Sub` 是项目自己的方法，返回 `int`：

```go
package sample

import "time"

type Clock struct{}

func (Clock) Sub(time.Time) int { return 1 }
func f(c Clock) int { return c.Sub(time.Now()) }
```

当前默认规则把最后一行改成 `return time.Until(c)`。这要求 `c` 是 `time.Time`，且结果是 `time.Duration`，与原始契约均不符。此问题与格式偏好无关，应先修复。

### 建议实施顺序与验收

1. **修复测试与只读检查。** 错误 `.want` 必须失败；`-d` 执行前后目标源码和模块文件内容一致。增加针对参数数量和上述重写误判的回归用例。
2. **修复类型加载和并发。** 支持的 Go 版本均通过全量测试；多文件 diff 通过 race 检查；同一输入产生相同顺序的输出。保留 AST/parser 的纯排版能力，同时明确类型规则的执行前提。
3. **拆清职责并测性能。** 纯排版不执行 tidy，不加载类型；import 分组沿用现有规则；语义现代化由独立入口或显式选项触发。名称与默认行为需在实施时一起更新 CLI/API/README，避免增加互相重叠的开关。用单文件、Git 变更文件集和全仓三种任务记录冷/热耗时与内存，确认收益。
4. **有需求时扩展。** 参考 `gci` 增加 `go.work` 多模块分组、多个 prefix；参考 `gofumpt` 补注释归属、幂等性与 gofmt 兼容测试。行长限制目前没有需求证据，暂缓。官方 `go fix` 已覆盖的规则逐条对照 fixtures，优先复用其执行方式；自有规则有实际差异时继续维护，不引入通用插件框架。

暂不新增生产依赖。若后续直接采用 `gofumpt`，需评估默认空行变化、与现有 `xast` 规则的冲突、固定版本的维护成本，再决定是否替换当前排版实现。类型相关自有规则可评估已有 `x/tools/go/analysis` 与 `analysistest`，先迁移一条已验证有缺陷的规则，无需一次重写整个工具。

## 本次验证记录

- `go version`：Go 1.27.2，darwin/arm64。
- `go build -buildvcs=false -o <临时路径> .`、`go vet ./...`：通过。
- `go test ./...`：失败于 `internal/gofmt` 的 `TestFormatter_Execute/app2`；导出数据解码错误如上。其他包本次测试通过，但使用 `xtest.CheckFile` 的输出断言存在上述漏洞。
- 临时副本中的错误 golden fixture：测试退出码 0，证明当前辅助函数未使差异失败。
- 隔离临时模块：复现 `-d` 修改 `go.mod`、局部对象与自定义方法误改、参数不足导致 panic。
- `go build -race` 后处理 50 个文件：检测到配置加载与 diff 收集的数据竞争。未据此推断其他并发路径均已检查。

所有实验在临时目录或临时仓库副本内进行；临时构建与探针已清理。本次仅新增调研报告，未修改生产代码、依赖或现有测试。

## 融合结果

2026-10-13，按用户澄清后的要求，将能力统一到 gorgeous 的默认格式化流程。删除此前的 `-style`、`-fix-imports`、`-section` 工具选择参数及对应 API 字段。

- 默认排版使用 gofumpt v0.12.0 的库接口，继续执行 gorgeous 的注释整理、代码简化与内置重写。
- 默认补删 import 使用 x/tools v0.52.0 的 goimports 库，在代码重写后修复导入。
- 默认按原有 `stc` 顺序使用 GCI v0.14.0 分组，工作区模块归入当前模块组；既有自定义分组策略继续生效。
- 默认使用 golangci/golines v0.16.0 的 `shorten` 库，目标宽度 120；`-max-len` 调整宽度，0 关闭。不修改 struct tags。
- `gorgeous fix` 保留为独立的 Go 工具链代码迁移入口，格式化不自动执行全量 `go fix`。已有内置重写按模块语言版本执行现代化规则。
- `-format-only` 不加载包类型，不执行代码重写；`-srcdir` 为 stdin 提供模块与 import 解析位置。

生产路径已移除自动 tidy。类型信息改为每次执行独立持有，文件计算并行，输出统一排序；路径去重避免重复处理同一文件。修复 golden 断言、已复现的函数/接收者误判及参数不足 panic。第三方 `strings` 测试的旧期望值原本与其注释不符，现改为保留调用。CI 的 OS matrix 已绑定实际 runner，staticcheck 固定为 v0.8.2。

依赖选择：gofumpt、GCI 使用 BSD-3-Clause，golines 使用 MIT；复用现有 x/tools，新增三项直接生产依赖。实际库调用不依赖外部 formatter 命令，不需要 CGO。GCI 引入 Zap/YAML/diff 依赖，golines 引入 decorated AST 和 structtags；未使用的 CLI 依赖已由 `go mod tidy` 移除。完整许可随二进制发布，见根目录 `THIRD_PARTY_LICENSES.txt`。

已验证 Go 1.26.0 和 1.27.2 全量测试、race、vet、staticcheck、默认流程幂等性、注释保留、生成文件跳过、只读模块检查，以及无参数 CLI 的 Git 文件选择、默认格式化、只读预览与输出编译；此前也验证了 stdin 与 go fix 预览/应用。govulncheck v1.9.0 未发现已知漏洞。

升级前性能样例（不代表融合后的默认流程）：本机 darwin/arm64，预构建 CLI，对一个无外部依赖文件执行 `-d`，预热一次后取三次中位数。默认模式约 28.63 ms，`-format-only` 约 6.52 ms。它们执行的规则不同，此结果说明跳过类型加载的成本收益，不是跨工具或大型仓库基准。
