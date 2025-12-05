# Noname Linux 包管理器重写计划

## 项目目标

使用 Zig 重写包管理器，实现以下目标：
1. **单一静态二进制** - 完全静态链接，无任何运行时依赖（包括 libc）
2. **架构拆分** - 构建和安装逻辑分离，为云端 CI/CD 做准备
3. **修复核心问题** - 解决数据库、sudo 依赖、文件删除等 bug
4. **更好的 C 库集成** - 静态链接 libarchive/zlib/xz/sqlite 等

## 核心约束

### 关键需求：救命稻草模式
```
场景：系统文件被误删（包括 libc、coreutils）
要求：pkg 二进制仍能正常工作
实现：完全静态链接，零外部依赖
```

### 架构设计：单 Binary + 子命令

```
pkg                    # 单一二进制 (~5-8MB)
├── pkg build <name>   # 构建包（云端/本地）
├── pkg install <name> # 安装包（本地）
├── pkg remove <name>  # 卸载包
├── pkg query [name]   # 查询包
├── pkg upgrade        # 升级包
└── pkg sync           # 同步远程索引
```

**内部模块化：**
```zig
// src/main.zig
const build = @import("commands/build.zig");
const install = @import("commands/install.zig");
const db = @import("lib/db.zig");
const archive = @import("lib/archive.zig");

pub fn main() !void {
    const args = std.process.argsAlloc();
    if (std.mem.eql(u8, args[1], "build")) {
        try build.run(args[2..]);
    } else if (std.mem.eql(u8, args[1], "install")) {
        try install.run(args[2..]);
    }
    // ...
}
```

## TODO 列表

### Phase 0: 项目准备 (3-5 天)

#### TODO 0.1: 技术验证
- [ ] 验证 Zig 静态链接 musl libc
  ```bash
  zig build-exe -target x86_64-linux-musl -static hello.zig
  ldd pkg  # 输出: not a dynamic executable
  ```
- [ ] 验证 C 库静态链接（libarchive/sqlite3/libcurl）
- [ ] 选择 TOML 解析库（zig-toml 或 C 库 tomlc99）
- [ ] 测试在无 libc 环境下运行

#### TODO 0.2: 项目骨架
- [ ] 创建 `pkg-zig/` 目录结构
  ```
  pkg-zig/
  ├── build.zig           # 构建配置（静态链接）
  ├── build.zig.zon       # 依赖声明
  ├── src/
  │   ├── main.zig        # 主入口，命令路由
  │   ├── lib/            # 核心库
  │   │   ├── pkgfile.zig # Pkgfile 解析
  │   │   ├── db.zig      # SQLite 数据库
  │   │   ├── archive.zig # tar/压缩（libarchive）
  │   │   ├── fetch.zig   # HTTP 下载（libcurl）
  │   │   ├── util.zig    # 工具函数
  │   │   └── types.zig   # 数据结构
  │   └── commands/       # 子命令
  │       ├── build.zig   # pkg build
  │       ├── install.zig # pkg install
  │       ├── remove.zig  # pkg remove
  │       ├── query.zig   # pkg query
  │       └── sync.zig    # pkg sync
  └── tests/
      └── integration/    # 集成测试
  ```
- [ ] 配置 `build.zig` 静态链接选项
  ```zig
  const exe = b.addExecutable(.{
      .name = "pkg",
      .target = b.resolveTargetQuery(.{
          .cpu_arch = .x86_64,
          .os_tag = .linux,
          .abi = .musl,
      }),
      .optimize = .ReleaseSafe,
      .link_libc = true,
      .linkage = .static,
  });
  exe.linkSystemLibrary("archive");
  exe.linkSystemLibrary("sqlite3");
  exe.linkSystemLibrary("curl");
  ```

### Phase 1: 核心库实现 (Week 1-2)

#### TODO 1.1: Pkgfile 解析器
- [ ] 实现 `lib/pkgfile.zig`
  ```zig
  pub const Pkgfile = struct {
      name: []const u8,
      version: []const u8,
      desc: []const u8,
      deps: [][]const u8,
      makedeps: [][]const u8,
      baks: [][]const u8,
      source: []Source,
      ext: ?[]const u8,
      build: []const u8,
      keepla: bool = false,
  };

  pub fn parse(path: []const u8, allocator: Allocator) !Pkgfile { ... }
  ```
- [ ] 支持变量展开（`$version`, `$name`）
- [ ] 测试：解析 `ports/bash/Pkgfile`，与 Go 版本输出对比
- [ ] 测试：解析所有 320 个包的 Pkgfile

#### TODO 1.2: 数据库层（SQLite）
- [ ] 实现 `lib/db.zig`
  ```zig
  pub const Database = struct {
      // 数据库路径
      path: []const u8,
      db: *sqlite3,

      pub fn init(path: []const u8) !Database { ... }
      pub fn installPackage(self: *Database, pkg: Package) !void { ... }
      pub fn removePackage(self: *Database, name: []const u8) !void { ... }
      pub fn queryPackage(self: *Database, name: []const u8) !?Package { ... }
      pub fn queryFile(self: *Database, path: []const u8) ![][]const u8 { ... }
  };
  ```
- [ ] 设计表结构（见 CLAUDE.md TODO 3）
  - `packages` 表
  - `dependencies` 表
  - `files` 表（全局文件所有权）
  - `package_files` 表（多对多关系）
- [ ] 实现事务支持
- [ ] 实现冲突检测（安装前检查文件是否被其他包占用）
- [ ] 测试：插入/查询/删除操作

#### TODO 1.3: Archive 处理（libarchive）
- [ ] 实现 `lib/archive.zig`
  ```zig
  pub fn extractTarGz(src: []const u8, dest: []const u8) !void { ... }
  pub fn createTarGz(src: []const u8, dest: []const u8) !void { ... }
  pub fn listFiles(tarball: []const u8) ![]FileInfo { ... }
  ```
- [ ] 支持格式：tar.gz, tar.xz, tar.zst
- [ ] 保留文件属性（uid/gid/mode/xattr）
- [ ] 测试：解压 Go 版本生成的 .tgz

#### TODO 1.4: HTTP 下载（libcurl）
- [ ] 实现 `lib/fetch.zig`
  ```zig
  pub fn download(url: []const u8, dest: []const u8, progress: ?*Progress) !void { ... }
  pub fn downloadWithChecksum(url: []const u8, dest: []const u8, sha256: []const u8) !void { ... }
  ```
- [ ] 支持代理（HTTP_PROXY/HTTPS_PROXY）
- [ ] 进度条（可选）
- [ ] 断点续传（可选）

#### TODO 1.5: 依赖解析
- [ ] 实现拓扑排序（`lib/deps.zig`）
- [ ] 循环依赖检测
- [ ] makedeps 和 deps 分离处理
- [ ] 测试：对比 Go 版本的依赖解析结果

### Phase 2: 子命令实现 (Week 2-3)

#### TODO 2.1: `pkg build` 命令
- [ ] 实现 `commands/build.zig`
  ```zig
  pub fn run(args: [][]const u8) !void {
      // 1. 解析 Pkgfile
      // 2. 解析 makedeps
      // 3. 安装 makedeps（临时）
      // 4. 下载源码
      // 5. 解压源码
      // 6. 执行 build 脚本（fakeroot）
      // 7. 自动处理（压缩 man、删除 .la）
      // 8. 打包 tarball
      // 9. 清理临时文件
      // 10. 卸载 makedeps
  }
  ```
- [ ] **关键改进：fakeroot 支持**
  - 使用 `fakeroot` 或 `fakechroot` 执行构建
  - 构建阶段不需要 root 权限
- [ ] 自动处理逻辑（from Go version）:
  - 压缩 man 页面
  - 删除 `.la` 文件（除非 `keepla=true`）
  - 复制 LICENSE 文件
  - 删除 `charset.alias` 等垃圾文件
- [ ] 支持 `--keep-build` 保留构建目录
- [ ] 支持 `--skip-deps` 跳过依赖安装
- [ ] 测试：构建简单包（如 `zlib`）
- [ ] 测试：构建复杂包（如 `firefox`）

#### TODO 2.2: `pkg install` 命令
- [ ] 实现 `commands/install.zig`
  ```zig
  pub fn run(args: [][]const u8) !void {
      // 1. 解析依赖
      // 2. 检查已安装
      // 3. 确认安装列表
      // 4. 开始数据库事务
      // 5. 执行 pre-install hooks
      // 6. 冲突检测（查询 files 表）
      // 7. 提取文件到 ROOT
      // 8. 更新数据库（全局文件追踪）
      // 9. 执行 post-install hooks
      // 10. 提交事务
  }
  ```
- [ ] **关键改进：文件冲突检测**
  ```sql
  -- 安装前检查
  SELECT package FROM package_files
  WHERE path = ? AND package != ?
  ```
- [ ] **关键改进：升级逻辑**
  ```zig
  // 只删除不再被任何包引用的文件
  for (old_files) |file| {
      const owners = db.queryFileOwners(file);
      if (owners.len == 1 and owners[0] == pkg_name) {
          // 只有当前包引用，可以删除
          os.remove(file);
      }
  }
  ```
- [ ] 支持 `--no-deps` 跳过依赖
- [ ] 支持 `--remote` 优先下载预编译包
- [ ] 测试：安装/升级/冲突场景

#### TODO 2.3: `pkg remove` 命令
- [ ] 实现 `commands/remove.zig`
  ```zig
  pub fn run(args: [][]const u8) !void {
      // 1. 检查反向依赖
      // 2. 确认删除列表
      // 3. 开始数据库事务
      // 4. 执行 pre-remove hooks
      // 5. 安全删除文件（检查引用计数）
      // 6. 清理空目录
      // 7. 更新数据库
      // 8. 执行 post-remove hooks
      // 9. 提交事务
  }
  ```
- [ ] **关键改进：安全删除**
  ```sql
  -- 只删除无其他引用的文件
  DELETE FROM files
  WHERE path = ?
  AND (SELECT COUNT(*) FROM package_files WHERE path = ?) = 0
  ```
- [ ] 测试：删除包后其他包仍正常
- [ ] 测试：删除共享文件的包（如 glibc）

#### TODO 2.4: `pkg query` 命令
- [ ] 实现 `commands/query.zig`
  ```zig
  pub fn run(args: [][]const u8) !void {
      // -a: 列出所有已安装包
      // -l: 列出包的文件列表（footprint）
      // -i: 显示包信息
      // -o <file>: 查询文件属于哪个包
      // -d <pkg>: 查询包的依赖
      // -r <pkg>: 查询谁依赖该包（反向依赖）
  }
  ```
- [ ] 输出格式兼容 Go 版本

#### TODO 2.5: `pkg sync` 命令（新功能）
- [ ] 实现 `commands/sync.zig`
  ```zig
  pub fn run(args: [][]const u8) !void {
      // 1. 下载远程包索引（JSON/TOML）
      // 2. 更新本地缓存
      // 3. 显示可升级的包
  }
  ```
- [ ] 远程索引格式：
  ```json
  {
    "packages": {
      "bash": {
        "version": "5.2.37",
        "url": "https://nas/noname/bash#5.2.37.tgz",
        "sha256": "..."
      }
    }
  }
  ```

### Phase 3: Hook 系统 (Week 3-4)

#### TODO 3.1: Hook 执行器
- [ ] 实现 `lib/hooks.zig`
  ```zig
  pub fn executeHooks(
      when: HookWhen,  // pre / post
      op: HookOp,      // install / remove
      files: [][]const u8,
      root: []const u8,
  ) !void { ... }
  ```
- [ ] 解析 `/etc/pkg/hooks/*.toml`
- [ ] 模式匹配（glob）
- [ ] chroot 执行
- [ ] 测试：fc-cache, gtk-update-icon-cache 等

#### TODO 3.2: Chroot 环境
- [ ] 实现 `lib/chroot.zig`
  ```zig
  pub fn setup(root: []const u8) !void {
      // mount /proc, /sys, /dev
      // bind mount /etc/resolv.conf
  }
  pub fn teardown(root: []const u8) !void {
      // umount all
  }
  pub fn execute(root: []const u8, cmd: [][]const u8) !void {
      // chroot and exec
  }
  ```
- [ ] 需要 root 权限（安装阶段）
- [ ] 错误处理（避免挂载泄漏）

### Phase 4: CI/CD 集成 (Week 4)

#### TODO 4.1: 远程仓库支持
- [ ] 实现远程包下载
  ```zig
  // pkg install firefox --remote
  // 1. 查询远程索引
  // 2. 下载预编译包
  // 3. 校验签名（可选）
  // 4. 安装
  ```
- [ ] 实现包上传（云端构建后）
  ```zig
  // pkg build --upload firefox
  // 1. 构建包
  // 2. 上传到 NAS/Forgejo
  // 3. 更新索引
  ```

#### TODO 4.2: Woodpecker 配置
- [ ] 创建 `.woodpecker/build.yml`
  ```yaml
  pipeline:
    detect-changes:
      image: alpine/git
      commands:
        - git diff --name-only HEAD^ HEAD > changed.txt

    build-packages:
      image: noname-builder:zig
      commands:
        - |
          for pkg in $(cat changed.txt | grep '^ports/' | cut -d/ -f2 | sort -u); do
            pkg build --upload $pkg
          done

    test-install:
      image: noname-builder:zig
      commands:
        - pkg install --remote firefox
        - firefox --version
  ```

#### TODO 4.3: 构建容器镜像
- [ ] 创建 `Dockerfile.builder`
  ```dockerfile
  FROM alpine:latest
  RUN apk add --no-cache zig libarchive-dev sqlite-dev curl-dev
  COPY pkg /usr/bin/pkg
  ```

### Phase 5: 测试和迁移 (Week 4-5)

#### TODO 5.1: 集成测试
- [ ] 测试场景：
  - [ ] 全新安装基础系统（lists_of_base）
  - [ ] 升级已安装包
  - [ ] 卸载包（检查残留文件）
  - [ ] 冲突检测
  - [ ] 循环依赖检测
  - [ ] Hook 执行
- [ ] 对比测试：Zig vs Go 版本
  - [ ] 数据库内容一致性
  - [ ] 文件系统状态一致性
  - [ ] 性能对比

#### TODO 5.2: 数据库迁移
- [ ] 实现 `pkg migrate` 命令
  ```zig
  pub fn migrate() !void {
      // 1. 读取 Go 版本数据库（diskv）
      // 2. 转换为 SQLite 格式
      // 3. 扫描文件系统补全 files 表
      // 4. 验证完整性
  }
  ```
- [ ] 测试迁移脚本（在测试环境）

#### TODO 5.3: 生产环境部署
- [ ] 准备回滚方案：
  ```bash
  # 保留 Go 版本二进制
  cp /bin/pkg /bin/pkg.go
  # 安装 Zig 版本
  cp pkg-zig/zig-out/bin/pkg /bin/pkg
  # 如果出问题，立即回滚
  cp /bin/pkg.go /bin/pkg
  ```
- [ ] 灰度测试：
  - [ ] 只在 chroot 环境测试
  - [ ] 安装少数包测试
  - [ ] 全量迁移
- [ ] 监控：
  - [ ] 记录所有操作日志
  - [ ] 对比新旧版本行为差异

#### TODO 5.4: 文档更新
- [ ] 更新 CLAUDE.md
- [ ] 更新 README.md
- [ ] 编写迁移指南
- [ ] 编写故障排查手册

### Phase 6: 后续优化（可选）

#### TODO 6.1: 性能优化
- [ ] 并行下载源码包
- [ ] 并行构建多个包（无依赖关系）
- [ ] 数据库查询优化（索引）

#### TODO 6.2: 高级功能
- [ ] 包签名和验证（GPG）
- [ ] 增量更新（delta 包）
- [ ] 本地缓存清理
- [ ] `pkg doctor` 诊断命令

#### TODO 6.3: Vim 高亮支持
- [ ] 创建 `contrib/vim/` 目录
- [ ] 实现 Pkgfile 语法文件
- [ ] 编写安装脚本

## 技术选型

### 核心依赖

| 功能 | 库 | 静态链接 | 理由 |
|------|-----|---------|------|
| TOML 解析 | tomlc99 | ✅ | 简单 C 库，易集成 |
| Archive | libarchive | ✅ | 支持所有格式，xattr 支持 |
| 数据库 | SQLite | ✅ | 单文件，事务支持 |
| HTTP | libcurl | ✅ | 功能完整，代理支持 |
| 压缩 | zlib/xz/zstd | ✅ | 通过 libarchive 间接使用 |

### 构建配置

```zig
// build.zig
pub fn build(b: *std.Build) void {
    const target = b.resolveTargetQuery(.{
        .cpu_arch = .x86_64,
        .os_tag = .linux,
        .abi = .musl,  // 静态链接 musl libc
    });

    const exe = b.addExecutable(.{
        .name = "pkg",
        .root_source_file = .{ .path = "src/main.zig" },
        .target = target,
        .optimize = .ReleaseSafe,
        .link_libc = true,
    });

    // 强制静态链接
    exe.linkage = .static;

    // 链接 C 库（静态）
    exe.linkSystemLibrary("archive");
    exe.linkSystemLibrary("sqlite3");
    exe.linkSystemLibrary("curl");
    exe.linkSystemLibrary("z");
    exe.linkSystemLibrary("lzma");
    exe.linkSystemLibrary("zstd");

    b.installArtifact(exe);
}
```

### 验证静态链接

```bash
zig build -Dtarget=x86_64-linux-musl -Doptimize=ReleaseSafe

# 验证
ldd zig-out/bin/pkg
# 预期输出: not a dynamic executable

# 验证大小（预期 5-8MB）
ls -lh zig-out/bin/pkg

# 验证在无 libc 环境运行
rm /lib/ld-musl-x86_64.so.1  # 危险！仅测试环境
./zig-out/bin/pkg --version   # 应该仍能运行
```

## 风险和缓解

### 风险 1：Zig 生态不成熟
**缓解：** 优先使用 C 库（libarchive/sqlite），而不是纯 Zig 实现

### 风险 2：数据库迁移失败
**缓解：**
- 迁移前完整备份 `/var/db/pkg`
- 实现双向验证脚本
- 保留 Go 版本作为回滚选项

### 风险 3：构建产物不兼容
**缓解：**
- Phase 2.1 先让 Zig 版本构建包
- 用 Go 版本安装测试
- 确认 tarball 格式兼容

### 风险 4：测试不充分
**缓解：**
- 保留 Go 版本在 `/bin/pkg.go`
- 灰度部署（先测试非关键包）
- 快速回滚机制

### 风险 5：静态链接失败
**缓解：**
- Phase 0.1 优先验证
- 准备 Plan B（动态链接，但保留 musl）

## 时间表

| 阶段 | 时间 | 可交付 | 里程碑 |
|------|------|--------|--------|
| Phase 0 | Week 1 | 项目骨架 + PoC | Pkgfile 解析器可用 |
| Phase 1 | Week 1-2 | 核心库 | 能构建简单包 |
| Phase 2 | Week 2-3 | 子命令 | 能安装/卸载包 |
| Phase 3 | Week 3-4 | Hook 系统 | 功能完整 |
| Phase 4 | Week 4 | CI/CD | 云端构建可用 |
| Phase 5 | Week 4-5 | 测试迁移 | 生产环境部署 |

**总计：4-5 周（有 AI 辅助）**

## 成功标准

### 必须达成：
- [x] 单一静态二进制，零外部依赖
- [ ] 通过所有集成测试（320 个包）
- [ ] 数据库无文件删除 bug
- [ ] 与 Go 版本功能等价

### 期望达成：
- [ ] 构建速度提升 20%+（静态链接 C 库）
- [ ] 二进制大小 < 10MB
- [ ] CI/CD 自动化构建

### 加分项：
- [ ] 性能超越 Go 版本
- [ ] 新功能（远程包缓存、签名验证）

## 开发环境要求

```bash
# 安装 Zig（最新版本）
wget https://ziglang.org/download/0.12.0/zig-linux-x86_64-0.12.0.tar.xz
tar xf zig-linux-x86_64-0.12.0.tar.xz
export PATH=$PWD/zig-linux-x86_64-0.12.0:$PATH

# 安装开发依赖（静态库）
pkg -S libarchive sqlite curl zlib xz zstd

# 验证
pkg --version
zig version
```

## 参考资源

- Go 版本代码：`/home/xhe/project/noname-linux/pkg/`
- Pkgfile 格式：`CLAUDE.md`
- 测试包：`/home/xhe/project/noname-linux/ports/`
- Zig 文档：https://ziglang.org/documentation/master/
- libarchive API：https://github.com/libarchive/libarchive/wiki

## 下一步行动

立即开始 **Phase 0.1: 技术验证**

```bash
cd /home/xhe/project/noname-linux
mkdir pkg-zig
cd pkg-zig

# 创建最小 PoC
cat > hello.zig << 'EOF'
const std = @import("std");
pub fn main() !void {
    std.debug.print("Hello from static Zig!\n", .{});
}
EOF

# 编译并验证
zig build-exe -target x86_64-linux-musl hello.zig
ldd hello  # 应该输出: not a dynamic executable
./hello
```

准备好开始了吗？
