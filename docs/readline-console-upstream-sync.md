# readline / console upstream sync

本文档用于定期把 `tui` 内嵌的 `readline`、`console` fork 同步到官方最新上游，并把结果沉淀回 `D:\Programing\go` 下两个长期维护库。

## 目标

- 官方上游通过 git merge 进入本地维护分支，避免手动覆盖导致历史断裂。
- `D:\Programing\go\readline` 和 `D:\Programing\go\console` 保持为可持续更新的长期 fork。
- `D:\Programing\go\chainreactors\tui\readline` 和 `D:\Programing\go\chainreactors\tui\console` 只接收已经 merge、测试通过的结果。
- 每次同步后能清楚回答：上游带来了什么、本地还保留了哪些 fork 改动、是否可安全接受。

## 固定路径

| 用途 | 路径 |
| --- | --- |
| tui 主仓库 | `D:\Programing\go\chainreactors\tui` |
| readline 长期库 | `D:\Programing\go\readline` |
| console 长期库 | `D:\Programing\go\console` |
| 临时 merge 根目录 | `D:\Programing\go\tui-upstream-merge` |
| readline merge worktree | `D:\Programing\go\tui-upstream-merge\readline` |
| console merge worktree | `D:\Programing\go\tui-upstream-merge\console` |

## 分支约定

| 仓库 | 官方远端 | 官方分支 | 本地维护分支 | 临时 merge 分支 |
| --- | --- | --- | --- | --- |
| `readline` | `origin=https://github.com/reeflective/readline` | `origin/master` | `iom` | `codex/tui-merge` |
| `console` | `origin=https://github.com/reeflective/console` | `origin/main` | `iom` | `codex/tui-merge` |
| `tui` | `origin=https://github.com/chainreactors/tui` | `origin/master` | feature branch | n/a |

`iom` 是长期维护分支。不要在官方 `master/main` 上放本地改动。

## 本地 fork 必须保留的能力

`readline`:

- module path: `github.com/chainreactors/tui/readline`
- `terminal` 包和 `NewShellWithTerminal`
- 自定义 terminal input/output/control、remote carrier 支持
- active terminal output/control、resize、raw mode 适配
- paste transformer、bracketed paste normalize、pending input 读取
- inline suggestion API 和渲染
- clipboard copy/paste 命令

`console`:

- module path: `github.com/chainreactors/tui/console`
- `replace github.com/chainreactors/tui/readline => ../readline`
- `NewWithTerminal`
- terminal-aware `Printf` / `TransientPrintf` / newline 输出
- paste reference API 和 `ResolvePasteReferences`
- `StartContext` 读取后解析 paste reference
- 导出的 `Console.Execute`
- completion panic recover、隐藏 `_carapace`、重置 Cobra/pflag 状态
- inline suggestion 对接 readline
- `Suggestion` 兼容类型

这些能力必须有回归测试覆盖。同步上游后，如果相关测试失败，优先判断是否误删/误改了本地 fork 能力，不要为了接受上游直接删除测试。

## 定期同步流程

建议每月一次，或在 tui release 前执行一次。

### 1. 更新远端信息

```powershell
cd D:\Programing\go\readline
git fetch origin --tags

cd D:\Programing\go\console
git fetch origin --tags

cd D:\Programing\go\chainreactors\tui
git fetch origin
```

### 2. 检查长期库是否干净

```powershell
cd D:\Programing\go\readline
git status --short --branch

cd D:\Programing\go\console
git status --short --branch
```

如果 `D:\Programing\go\readline` 或 `D:\Programing\go\console` 有未提交改动，不要直接在原目录 merge。使用下面的 worktree 流程，避免覆盖用户工作。

### 3. 准备临时 merge worktree

如果旧 worktree 存在，先确认没有未保存内容，再删除目录并 prune：

```powershell
cd D:\Programing\go\readline
git worktree list
git worktree prune

cd D:\Programing\go\console
git worktree list
git worktree prune
```

创建新的 merge worktree：

```powershell
New-Item -ItemType Directory -Force D:\Programing\go\tui-upstream-merge | Out-Null

cd D:\Programing\go\readline
git worktree add -B codex/tui-merge D:\Programing\go\tui-upstream-merge\readline iom

cd D:\Programing\go\console
git worktree add -B codex/tui-merge D:\Programing\go\tui-upstream-merge\console iom
```

### 4. 把当前 tui 内嵌库同步为 merge 起点

这一步只用于让长期库准确代表当前 `tui` 内嵌状态，然后再 merge 官方上游。

先 dry-run：

```powershell
robocopy D:\Programing\go\chainreactors\tui\readline D:\Programing\go\tui-upstream-merge\readline /MIR /XD .git .idea /L
robocopy D:\Programing\go\chainreactors\tui\console D:\Programing\go\tui-upstream-merge\console /MIR /XD .git .idea /XF example_linux_amd64 example_windows_amd64.exe /L
```

确认输出符合预期后执行：

```powershell
robocopy D:\Programing\go\chainreactors\tui\readline D:\Programing\go\tui-upstream-merge\readline /MIR /XD .git .idea
if ($LASTEXITCODE -gt 7) { exit $LASTEXITCODE } else { $global:LASTEXITCODE = 0 }

robocopy D:\Programing\go\chainreactors\tui\console D:\Programing\go\tui-upstream-merge\console /MIR /XD .git .idea /XF example_linux_amd64 example_windows_amd64.exe
if ($LASTEXITCODE -gt 7) { exit $LASTEXITCODE } else { $global:LASTEXITCODE = 0 }
```

提交当前 tui fork baseline：

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
git add -A
git commit -m "sync tui readline fork state"

cd D:\Programing\go\tui-upstream-merge\console
git add -A
git commit -m "sync tui console fork state"
```

如果没有变化，`git commit` 会提示 nothing to commit，可以继续下一步。

### 5. merge 官方上游

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
git merge --no-edit origin/master

cd D:\Programing\go\tui-upstream-merge\console
git merge --no-edit origin/main
```

如果有冲突，处理原则：

- 默认接受官方最新结构和重构。
- 只重新保留本文档列出的 fork 能力。
- 不保留 `.idea`、本地二进制、临时文件。
- 不为了过冲突而回退上游测试、并发修复、signal 修复、display/refactor。

检查冲突是否清完：

```powershell
git diff --name-only --diff-filter=U
rg -n "<<<<<<<|=======|>>>>>>>" -S .
```

注意：`rg` 可能匹配普通字符串里的 `=======`，例如帮助文本。以 `git diff --name-only --diff-filter=U` 为空为准。

### 6. 格式化、整理依赖、测试

`readline`:

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
$files = rg --files -g '*.go'
if ($files) { gofmt -w $files }
go mod tidy
go test ./...
```

`console`:

```powershell
cd D:\Programing\go\tui-upstream-merge\console
$files = rg --files -g '*.go'
if ($files) { gofmt -w $files }
go mod tidy
go test ./...
```

重点关注 fork 回归测试：

- `readline/fork_test.go`
- `readline/terminal/fork_test.go`
- `console/fork_test.go`
- `console/paste_test.go`

这些测试用于防止上游同步时删掉本地 terminal、paste、inline suggestion、Execute 等功能。

依赖原则：

- 不因为 `go mod tidy` 无意升级依赖。
- 如果只是工具链解析导致 `golang.org/x/sys`、`golang.org/x/exp` 升级，优先固定回官方上游版本。
- 如果本地 fork 新能力确实需要新增依赖，记录原因。

### 7. 提交 merge 结果

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
git add -A
git commit -m "merge upstream readline"

cd D:\Programing\go\tui-upstream-merge\console
git add -A
git commit -m "merge upstream console"
```

如果已经提交后又修了依赖或测试，可用：

```powershell
git add -A
git commit --amend --no-edit
```

### 8. 更新长期 `iom` 分支

只有当原目录工作区干净时执行。

```powershell
cd D:\Programing\go\readline
git status --short --branch
git switch iom
git merge --ff-only codex/tui-merge

cd D:\Programing\go\console
git status --short --branch
git switch iom
git merge --ff-only codex/tui-merge
```

如果原目录不干净，先不要更新 `iom`。保留 worktree merge commit，等用户改动处理完后再 fast-forward。

### 9. copy 回 tui

先从 `master` 新建 PR 分支：

```powershell
cd D:\Programing\go\chainreactors\tui
git switch master
git pull --ff-only
git switch -c chore/sync-readline-console-upstream
```

先 dry-run：

```powershell
robocopy D:\Programing\go\tui-upstream-merge\readline D:\Programing\go\chainreactors\tui\readline /MIR /XD .git .idea /L
robocopy D:\Programing\go\tui-upstream-merge\console D:\Programing\go\chainreactors\tui\console /MIR /XD .git .idea /XF example_linux_amd64 example_windows_amd64.exe /L
```

确认后执行：

```powershell
robocopy D:\Programing\go\tui-upstream-merge\readline D:\Programing\go\chainreactors\tui\readline /MIR /XD .git .idea
if ($LASTEXITCODE -gt 7) { exit $LASTEXITCODE } else { $global:LASTEXITCODE = 0 }

robocopy D:\Programing\go\tui-upstream-merge\console D:\Programing\go\chainreactors\tui\console /MIR /XD .git .idea /XF example_linux_amd64 example_windows_amd64.exe
if ($LASTEXITCODE -gt 7) { exit $LASTEXITCODE } else { $global:LASTEXITCODE = 0 }
```

### 10. 在 tui 中测试和提交

```powershell
cd D:\Programing\go\chainreactors\tui
go test ./...
git status --short
git add readline console
git commit -m "sync readline and console with upstream"
```

如果 `go.mod` / `go.sum` 因子模块变化需要更新，再单独检查后加入 commit。

### 11. 创建 PR

```powershell
git push -u origin chore/sync-readline-console-upstream
```

PR 描述建议包含：

- 官方 upstream commit：
  - readline: `git -C D:\Programing\go\tui-upstream-merge\readline rev-parse --short origin/master`
  - console: `git -C D:\Programing\go\tui-upstream-merge\console rev-parse --short origin/main`
- 本地 merge commit：
  - readline: `git -C D:\Programing\go\tui-upstream-merge\readline rev-parse --short HEAD`
  - console: `git -C D:\Programing\go\tui-upstream-merge\console rev-parse --short HEAD`
- 测试结果：
  - `go test ./...` in readline
  - `go test ./...` in console
  - `go test ./...` in tui
- 保留的本地 fork 能力。
- Go version 变化。如果官方子模块 `go.mod` 提升了 `go` directive，明确说明是否接受。

## 快速差异检查

查看 fork 与官方最新还差哪些文件：

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
git diff --shortstat origin/master HEAD
git diff --name-status origin/master HEAD

cd D:\Programing\go\tui-upstream-merge\console
git diff --shortstat origin/main HEAD
git diff --name-status origin/main HEAD
```

查看本地保留提交和官方提交：

```powershell
cd D:\Programing\go\tui-upstream-merge\readline
git log --oneline --decorate --left-right --cherry-pick HEAD...origin/master

cd D:\Programing\go\tui-upstream-merge\console
git log --oneline --decorate --left-right --cherry-pick HEAD...origin/main
```

## 当前已知状态

2026-06-27 已完成一次 merge 验证：

- `readline`
  - official latest: `088046b`
  - merge result: `916e20f`
  - test: `go test ./...` passed
- `console`
  - official latest: `7002774`
  - merge result: `92517a0`
  - test: `go test ./...` passed

注意：当前官方 `readline` / `console` 子模块为 `go 1.25.0`，而 `tui` 根模块是 `go 1.24.2`。copy 回 `tui` 并发 PR 前，需要确认是否接受子模块 Go 版本要求。
