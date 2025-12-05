# Noname Linux 项目文档

这是一个自制的 Linux 发行版，基于 musl libc，专注轻量化和个人定制。

## 项目目录结构

```
noname-linux/
├── pkg/                    # Go 编写的包管理器源代码
│   ├── main.go            # CLI 入口，参数解析，配置加载
│   ├── pkgfile.go         # Pkgfile 解析器，处理包定义
│   ├── pkgdb.go           # 数据库接口（diskv 封装）
│   ├── download.go        # 下载源码包（支持 http/git/等）
│   ├── extract.go         # 解压源码包
│   ├── build.go           # 构建包（编译 → 打包 .tgz）
│   ├── install.go         # 安装包（依赖解析 → 文件提取）
│   ├── remove.go          # 卸载包（反向依赖检查 → 文件删除）
│   ├── query.go           # 查询包信息
│   ├── hook.go            # 安装/卸载前后的 hook 脚本执行
│   ├── chroot.go          # chroot 环境设置（用于 hook）
│   └── util.go            # 工具函数
│
├── ports/                  # 活跃的包定义（约 320 个）
│   ├── bash/Pkgfile       # 每个包一个目录
│   ├── linux/Pkgfile      # Pkgfile 定义包的构建方式
│   └── ...
│
├── archived/               # 归档的包（约 719 个）
│   └── ...                # 不再维护但保留的包
│
├── scripts/                # 自动化脚本
│   ├── package            # 批量构建基础系统
│   ├── lists_of_base      # 基础包列表
│   ├── lists_of_dev       # 开发工具包列表
│   ├── relocating         # 迁移脚本
│   └── Dockerfile         # 容器构建环境
│
└── newpkg/                 # 新包开发临时目录
```

## 包管理器架构

### 1. 命令行接口（类 Pacman 风格）

**主操作（main action）：**
- `-M` - 构建包（Make）：下载 → 解压 → 编译 → 打包
- `-S` - 安装包（Sync）：解析依赖 → 安装包
- `-R` - 卸载包（Remove）：检查反向依赖 → 删除文件
- `-Q` - 查询包（Query）：查看已安装包、footprint、依赖
- `-U` - 下载包源码（Update）
- `-E` - 解压包源码（Extract）

**子操作（sub action）：**
- `-d` - 忽略依赖
- `-k` - 构建时保留 src/pkg 目录
- `-r` - 升级已安装包
- `-c` - 更新 .checksum
- `-z` - 不标记为用户安装
- 等等...

**配置文件（/etc/pkg/config）：**
```toml
http_proxy = "..."
arch = "x86_64"
target = "x86_64-linux-musl"
root = "/"                    # 安装根目录
buildroot = "/tmp/pkg"        # 构建临时目录
database = "/var/db/pkg"      # 包数据库路径
hooks = "/etc/pkg/hooks"      # hook 脚本目录
ports = "/path/to/ports"      # Pkgfile 仓库路径
CFLAGS = "..."
makeflags = 16                # 并行编译线程数
ccache = true
```

### 2. 包定义格式（Pkgfile）

使用 TOML 格式定义包：

```toml
version = "5.2.37"
desc = "An sh-compatible command language interpreter"
makedeps = ["ncurses", "readline"]  # 编译时依赖
deps = ["ncurses", "readline", "musl"]  # 运行时依赖
baks = ["etc/bashrc"]  # 升级时备份的文件

[[source]]
url = "http://ftp.gnu.org/gnu/bash/bash-$version.tar.gz"

[[source]]
url = "bashrc"  # 本地文件

ext = '''
  cd bash-$version
  patch -p0 -i "$srcdir/some.patch"
'''

build = '''
  cd bash-$version
  ./configure --prefix=/
  make
  make DESTDIR="$pkgdir" install
  install -D -m644 "$srcdir"/bashrc "$pkgdir"/etc/bashrc
'''
```

**变量：**
- `$name`, `$version` - 包名和版本
- `$srcdir` - 源码目录（`/tmp/pkg/<name>/src`）
- `$pkgdir` - 安装目录（`/tmp/pkg/<name>/pkg`）
- `$source` - 所有源文件名（换行分隔）

### 3. 数据库设计

**技术栈：** diskv（基于文件系统的键值存储 + zlib 压缩）

**Pkgdb 结构：**
```go
type Pkgdb struct {
    Name      string
    Version   string
    Deps      []string   // 运行时依赖
    Desc      string
    Baks      []string   // 备份文件列表
    Footprint []Fileinfo // 已安装文件列表
    User      bool       // 是否用户主动安装（用于自动清理）
}

type Fileinfo struct {
    Path string  // 相对于 ROOT 的路径
    Uid  int
    Gid  int
    Mode int64
    Size int64
    Type byte    // tar.TypeDir / tar.TypeReg / ...
}
```

**存储位置：** `/var/db/pkg/<pkgname>` （TOML 格式，zlib 压缩）

### 4. 核心工作流

#### 构建包（`pkg -M <name>`）

1. **解析依赖** - 递归查询 makedeps + deps
2. **安装 makedeps** - 临时安装编译依赖
3. **下载源码** - 支持 http/https/git/file 等协议
4. **校验** - 对比 .checksum 文件
5. **解压** - 根据 `ext` 字段处理源码
6. **构建** - 在 chroot 外执行 `build` 脚本
7. **自动处理**：
   - 压缩 man 页面（gzip）
   - 删除 `.la` 文件（除非 `keepla=true`）
   - 复制 LICENSE 文件到 `/share/licenses/<name>/`
   - 删除 `charset.alias`, `perllocal.pod` 等垃圾文件
8. **打包** - 生成 `<name>#<version>.tgz` tarball
9. **清理** - 删除 src/pkg 目录（除非 `-k`）
10. **自动卸载 makedeps** - 删除不再需要的编译依赖

#### 安装包（`pkg -S <name>`）

1. **解析依赖** - 拓扑排序，检测循环依赖
2. **确认** - 显示待安装包列表
3. **执行 pre-install hooks**
4. **提取文件**：
   - 从 tarball 提取到 ROOT
   - 处理备份文件（`.bak` 后缀）
   - 保留 xattr（如 capabilities）
5. **执行 post-install hooks** - 更新缓存、重载服务等
6. **更新数据库** - 写入 Pkgdb
7. **删除旧文件** - 如果是升级，删除不在新 footprint 中的文件

#### 卸载包（`pkg -R <name>`）

1. **检查反向依赖** - 查找依赖该包的其他包
2. **确认** - 显示待删除包列表
3. **执行 pre-remove hooks**
4. **删除文件**：
   - 按 footprint 删除文件
   - 删除空目录（递归向上）
   - 删除 `.bak` 备份文件
5. **执行 post-remove hooks**
6. **删除数据库记录**

### 5. Hook 系统

Hook 定义在 `/etc/pkg/hooks/<name>.toml`：

```toml
pattern = ["share/glib-2.0/schemas/*"]
when = ["post"]          # pre / post
operation = ["install"]  # install / remove
noarg = true             # 不传递匹配文件列表

script = '''
  glib-compile-schemas /share/glib-2.0/schemas
'''
```

**执行环境：** chroot 到 ROOT，传入 `$ROOT`, `$ARCH` 等环境变量

**常见 Hook：**
- 更新字体缓存（fc-cache）
- 更新图标缓存（gtk-update-icon-cache）
- 重新生成 initramfs（mkinitcpio）
- 更新 MIME 数据库
- 重载 systemd 单元

### 6. 特性与限制

**优点：**
- 依赖解析准确（拓扑排序 + 循环检测）
- 支持文件锁（避免并发构建冲突）
- Hook 系统灵活
- 支持交叉编译（ARCH, TARGET, OTARGETS）
- 自动处理常见打包任务

**当前问题：**
- **需要 sudo 运行** - 构建和系统操作混合
- **文件删除 bug** - 安装/卸载时可能删不干净或删错文件
- **数据库简陋** - diskv 不支持事务，易损坏
- **手动管理产物** - 编译的 `.tgz` 容易丢失

## 已知 Bug 和待改进

### 1. 文件删除问题（install.go / remove.go）

**问题描述：**
- 升级包时：可能遗留旧版本文件
- 卸载包时：可能删除被其他包共享的文件
- 空目录清理：逻辑不完善，可能误删或遗留

**根本原因：**
- Footprint 只记录自己安装的文件，没有全局文件所有权追踪
- 删除时没有检查文件是否被其他包引用
- 空目录删除算法（remove.go:103-117）过于简单

### 2. Sudo 依赖

**问题：**
- 构建需要 root（chroot hook 需要）
- 文件操作需要 root（写入 /usr, /etc 等）
- 增加安全风险

**改进方向：**
- 参考 Arch Linux 的 makepkg：
  - 构建阶段使用普通用户（fakeroot）
  - 仅安装阶段需要 root
  - 分离 `pkg-build` 和 `pkg-install`

### 3. 数据库可靠性

**问题：**
- diskv 无事务支持
- 异常退出可能损坏数据库
- 无版本管理和迁移机制

**改进方向：**
- 使用 SQLite（事务 + 索引 + 完整性约束）
- 或使用 BoltDB（嵌入式 KV，ACID）

## TODO

### TODO 1: Woodpecker CI/CD 集成

**目标：** 自动化构建，解决编译产物易丢失的问题

**实施步骤：**

1. **创建构建容器镜像**
   - 基于当前系统制作 Docker 镜像
   - 包含 `pkg` 工具和基础编译环境

2. **编写 Woodpecker 配置**
   - 监听 `ports/*/Pkgfile` 变更
   - 触发增量构建（只构建变更的包）
   - 并行构建无依赖的包

3. **产物管理**
   - 上传 `.tgz` 到 NAS/Forgejo Release
   - 维护包索引（name → URL 映射）
   - 支持版本历史

4. **本地集成**
   - `pkg` 增加 `--remote-fallback` 参数
   - 优先下载预编译包，失败再本地构建

**预期效果：**
- 每次 push 自动构建变更的包
- 本地安装速度大幅提升
- 构建历史可追溯

### TODO 2: 剥离 sudo 依赖（参考 makepkg）

**目标：** 构建阶段使用普通用户，提高安全性

**改进方案：**

1. **分离构建和安装**
   - `pkg-build` - 不需要 root，生成 `.tgz`
   - `pkg-install` - 需要 root，安装文件

2. **使用 fakeroot**
   - 构建时伪造 root 权限
   - 在 fakeroot 环境中执行 `build()` 脚本

3. **调整 hook 逻辑**
   - Pre-install hook 仍需 chroot
   - Build 阶段不执行需要 root 的操作

4. **目录权限**
   - buildroot 使用 `$HOME/.cache/pkg/build`
   - 用户可写，无需 root

**代码变更：**
- `build.go` - 添加 fakeroot 包装
- `install.go` - 检查 root 权限
- `main.go` - 根据操作检查权限

### TODO 3: 重写数据库和安装/卸载逻辑

**目标：** 修复文件删除 bug，提高可靠性

**问题根源：**
1. **文件所有权不明确**
   - 当前：每个包记录自己的 footprint
   - 问题：多个包可能安装相同文件（如 `/share/info/dir`）

2. **升级逻辑缺陷**（install.go:289-293）
   ```go
   // 删除旧文件：只检查新 footprint 中不存在
   // 但没考虑该文件可能被其他包引用
   for _, x := range pkgdb.Footprint {
       if !t[x.Path] {
           os.Remove(filepath.Join(cfg.ROOT, x.Path))
       }
   }
   ```

3. **卸载逻辑缺陷**（remove.go:91-126）
   - 直接删除 footprint 中的文件
   - 没有检查该文件是否被其他包引用

### TODO 4: nvchecker

**改进方案：**

#### 3.1 重写数据库层

**新结构：** 使用 SQLite

```sql
CREATE TABLE packages (
    name TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    desc TEXT,
    user INTEGER DEFAULT 0  -- 是否用户安装
);

CREATE TABLE dependencies (
    package TEXT,
    dependency TEXT,
    type TEXT,  -- 'runtime' or 'make'
    FOREIGN KEY (package) REFERENCES packages(name)
);

CREATE TABLE files (
    path TEXT PRIMARY KEY,
    uid INTEGER,
    gid INTEGER,
    mode INTEGER,
    size INTEGER,
    type INTEGER,
    hash TEXT  -- 内容哈希，用于检测冲突
);

CREATE TABLE package_files (
    package TEXT,
    path TEXT,
    FOREIGN KEY (package) REFERENCES packages(name),
    FOREIGN KEY (path) REFERENCES files(path)
);

CREATE TABLE backups (
    package TEXT,
    path TEXT,
    FOREIGN KEY (package) REFERENCES packages(name)
);
```

**优势：**
- 全局文件所有权追踪（可查询 `/usr/bin/bash` 属于哪些包）
- 支持事务（安装失败可回滚）
- 高效查询（索引加速依赖解析）

#### 3.2 重写安装逻辑

```go
func pkg_install(pkgs []string) error {
    tx := db.Begin()  // 开始事务
    defer tx.Rollback()

    // 1. 解析所有待安装文件
    allFiles := map[string]*FileInfo{}
    for _, pkg := range pkgs {
        footprint := pkg.Footprint()
        for _, f := range footprint {
            if existing, ok := allFiles[f.Path]; ok {
                // 冲突检测
                if existing.Hash != f.Hash {
                    return fmt.Errorf("file conflict: %s", f.Path)
                }
            }
            allFiles[f.Path] = f
        }
    }

    // 2. 检查与已安装包的冲突
    for path := range allFiles {
        owners := tx.Query("SELECT package FROM package_files WHERE path = ?", path)
        for _, owner := range owners {
            if !contains(pkgs, owner) {
                // 已被其他包安装，检查内容是否一致
                existing := tx.QueryFile(path)
                if existing.Hash != allFiles[path].Hash {
                    return fmt.Errorf("file conflict with %s: %s", owner, path)
                }
            }
        }
    }

    // 3. 提取文件
    for _, pkg := range pkgs {
        pkg_cpfiles(pkg)
    }

    // 4. 更新数据库
    for _, pkg := range pkgs {
        // 删除旧版本记录
        tx.Exec("DELETE FROM packages WHERE name = ?", pkg.Name)
        tx.Exec("DELETE FROM package_files WHERE package = ?", pkg.Name)

        // 插入新记录
        tx.Exec("INSERT INTO packages ...", pkg)
        for _, f := range pkg.Footprint() {
            tx.Exec("INSERT OR IGNORE INTO files ...", f)
            tx.Exec("INSERT INTO package_files ...", pkg.Name, f.Path)
        }
    }

    tx.Commit()  // 提交事务
    return nil
}
```

#### 3.3 重写卸载逻辑

```go
func pkg_remove(pkgs []string) error {
    tx := db.Begin()
    defer tx.Rollback()

    // 1. 收集待删除文件
    filesToRemove := map[string]bool{}
    for _, pkg := range pkgs {
        files := tx.Query("SELECT path FROM package_files WHERE package = ?", pkg)
        for _, f := range files {
            filesToRemove[f] = true
        }
    }

    // 2. 检查文件是否被其他包引用
    for path := range filesToRemove {
        owners := tx.Query(`
            SELECT package FROM package_files
            WHERE path = ? AND package NOT IN (?)
        `, path, pkgs)

        if len(owners) > 0 {
            // 被其他包引用，不删除
            delete(filesToRemove, path)
        }
    }

    // 3. 删除文件（按深度倒序，先删文件再删目录）
    sortedFiles := sortByDepth(filesToRemove)
    for _, path := range sortedFiles {
        os.Remove(filepath.Join(cfg.ROOT, path))
    }

    // 4. 清理空目录
    cleanEmptyDirs(cfg.ROOT)

    // 5. 更新数据库
    for _, pkg := range pkgs {
        tx.Exec("DELETE FROM packages WHERE name = ?", pkg)
        tx.Exec("DELETE FROM package_files WHERE package = ?", pkg)
        // files 表保留（可能被其他包引用）
        tx.Exec("DELETE FROM files WHERE path NOT IN (SELECT path FROM package_files)")
    }

    tx.Commit()
    return nil
}
```

**关键改进：**
- **安全删除** - 只删除无其他引用的文件
- **冲突检测** - 安装前检查文件冲突
- **事务保护** - 操作失败可回滚
- **完整性约束** - 数据库强制引用完整性

### TODO 4: Vim 语法高亮支持

**目标：** 为 Pkgfile 添加 Vim 语法高亮，特别是 shell 脚本部分

**问题描述：**
- Pkgfile 是 TOML 格式，基础高亮可用
- 但 `ext` 和 `build` 字段中的 shell 脚本（三引号字符串）没有高亮
- 影响编辑体验，容易出错

**实施方案：**

#### 4.1 创建 Vim 语法文件

创建 `~/.vim/after/syntax/toml.vim`（或系统路径）：

```vim
" Pkgfile 语法增强 - 为 TOML 中的 shell 脚本添加高亮
" 检测 ext/build 字段的三引号字符串

" 定义 shell 脚本区域
syntax region pkgfileShellScript
  \ matchgroup=tomlString
  \ start=/\v(ext|build)\s*\=\s*'''/
  \ end=/'''/
  \ contains=@Shell
  \ keepend

" 如果上面的规则不够精确，可以使用更严格的匹配
syntax region pkgfileShellScript
  \ matchgroup=tomlString
  \ start=/\v^(ext|build)\s*\=\s*'''$/
  \ end=/^'''$/
  \ contains=@Shell
  \ keepend

" 链接到 Shell 语法组
hi def link pkgfileShellScript Normal
```

#### 4.2 自动检测 Pkgfile

创建 `~/.vim/ftdetect/pkgfile.vim`：

```vim
" 自动识别 Pkgfile 为 toml 类型，并加载增强语法
au BufRead,BufNewFile Pkgfile set filetype=toml
au BufRead,BufNewFile */ports/*/Pkgfile set filetype=toml
```

#### 4.3 完整的语法文件（可选）

如果需要更完整的控制，创建独立的 `pkgfile.vim` 语法文件：

```vim
" Vim syntax file
" Language: Pkgfile (Noname Linux package definition)
" Maintainer: xw897002528@gmail.com

if exists("b:current_syntax")
  finish
endif

" 加载 TOML 语法作为基础
runtime! syntax/toml.vim
unlet b:current_syntax

" 扩展：shell 脚本高亮
syntax include @Shell syntax/sh.vim

" 匹配 ext = ''' ... '''
syntax region pkgfileExt
  \ matchgroup=tomlKey
  \ start=/^\s*ext\s*=\s*'''/
  \ end=/'''/
  \ contains=@Shell
  \ keepend

" 匹配 build = ''' ... '''
syntax region pkgfileBuild
  \ matchgroup=tomlKey
  \ start=/^\s*build\s*=\s*'''/
  \ end=/'''/
  \ contains=@Shell
  \ keepend

" 高亮 Pkgfile 特有关键字
syntax keyword pkgfileKeyword version desc makedeps deps baks source ext build
syntax keyword pkgfileKeyword keepla nostrip
syntax keyword pkgfileVariable $name $version $srcdir $pkgdir $source

hi def link pkgfileKeyword Keyword
hi def link pkgfileVariable Identifier

let b:current_syntax = "pkgfile"
```

#### 4.4 安装方式

**方式一：手动安装**
```bash
mkdir -p ~/.vim/{syntax,ftdetect,after/syntax}
# 复制语法文件到对应位置
```

**方式二：项目内分发**
在 noname-linux 仓库添加：
```
contrib/vim/
├── syntax/pkgfile.vim
├── ftdetect/pkgfile.vim
└── install.sh
```

安装脚本：
```bash
#!/bin/bash
# contrib/vim/install.sh
VIM_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/vim"
mkdir -p "$VIM_DIR"/{syntax,ftdetect}
cp syntax/pkgfile.vim "$VIM_DIR"/syntax/
cp ftdetect/pkgfile.vim "$VIM_DIR"/ftdetect/
echo "Pkgfile syntax installed to $VIM_DIR"
```

#### 4.5 Neovim 支持

Neovim 用户可以使用 TreeSitter（更精确）：

```lua
-- ~/.config/nvim/after/ftplugin/toml.lua
-- 检测 Pkgfile 并注入 shell 高亮
if vim.fn.expand('%:t') == 'Pkgfile' then
  -- 使用 TreeSitter 注入 shell 语法
  vim.treesitter.language.register('bash', 'toml')
end
```

或创建自定义 TreeSitter 查询：
```scheme
; ~/.config/nvim/after/queries/toml/injections.scm
; 为 Pkgfile 的 ext/build 字段注入 bash 高亮
((pair
  key: (bare_key) @_key
  value: (string) @injection.content)
 (#match? @_key "^(ext|build)$")
 (#set! injection.language "bash"))
```

**预期效果：**
- `ext` 和 `build` 中的 shell 命令有语法高亮
- 支持 vim 和 neovim
- 提升 Pkgfile 编辑体验

## 开发注意事项

1. **测试环境** - 始终在 chroot 或容器中测试包管理操作
2. **备份数据库** - 修改数据库逻辑前备份 `/var/db/pkg`
3. **增量迁移** - TODO 3 需要数据库迁移脚本
4. **兼容性** - 保持 Pkgfile 格式向后兼容
5. **日志** - 使用 `log.Debugf` 记录关键操作

## 相关资源

- 包仓库：`/home/xhe/project/noname-linux/ports`
- 数据库：`/var/db/pkg` (当前), `~/.local/share/pkg/db.sqlite` (TODO 3)
- 配置：`/etc/pkg/config`
- 构建缓存：`/tmp/pkg` (当前), `~/.cache/pkg` (TODO 2)
- CI/CD：Woodpecker + Forgejo (TODO 1)

## 维护者

xw897002528@gmail.com
