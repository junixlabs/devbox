# devbox — Tầm nhìn sản phẩm

> "Biến bất kỳ máy Linux nào thành dev environment sẵn sàng dùng trong 1 lệnh — không cần cloud, không cần DevOps."

---

## Mục lục

1. [Bối cảnh ra đời](#bối-cảnh-ra-đời)
2. [Vấn đề cần giải quyết](#vấn-đề-cần-giải-quyết)
3. [Nghiên cứu giải pháp hiện có](#nghiên-cứu-giải-pháp-hiện-có)
4. [Product Vision](#product-vision)
5. [Công dụng lớn nhất](#công-dụng-lớn-nhất)
6. [Multi-agent collaboration](#multi-agent-collaboration-killer-feature)
7. [Target Users](#target-users)
8. [Tại sao bây giờ](#tại-sao-bây-giờ)
9. [Architecture](#architecture)
10. [Tech Decisions](#tech-decisions)
11. [Roadmap](#roadmap)
12. [Nguyên tắc thiết kế](#nguyên-tắc-thiết-kế)
13. [Metrics thành công](#metrics-thành-công)

---

## Bối cảnh ra đời

### Câu chuyện bắt đầu từ 4GB

Developer dùng MacBook Air M2 256GB — con máy này đẹp, nhẹ, pin trâu, nhưng có 1 vấn đề
chết người: **ổ đĩa chỉ còn 4GB free**.

Phân tích dung lượng:
- **Docker Desktop**: ~15GB (images, volumes, build cache)
- **Project files**: ~24GB (nhiều repo, node_modules, vendor, build artifacts)
- **App cache**: ~20GB+ (Xcode cache, Homebrew, npm cache, pip cache)
- Tổng cộng gần **60GB** chỉ cho dev tools trên con máy 256GB

Đây không phải vấn đề cá nhân. Đây là vấn đề cấu trúc: **laptop ngày càng mỏng nhẹ,
SSD không thể nâng cấp, nhưng dev environment ngày càng nặng.**

### Hardware có sẵn nhưng chưa tận dụng

Trong khi đó, có 1 máy Ubuntu desktop (`dev1`) luôn mở:
- **CPU**: Intel i5-11400 (6 cores / 12 threads)
- **RAM**: 16GB DDR4
- **SSD**: 457GB (còn trống ~300GB)
- **Kết nối**: Tailscale mesh network, ping ~5ms từ MacBook

Máy này mạnh hơn MacBook về CPU multi-thread, có nhiều storage hơn gấp đôi,
và luôn online. Nhưng nó chỉ đang chạy vài service nhỏ.

### Thử nghiệm thủ công

Bước đầu thử nghiệm remote development bằng cách kết hợp:

1. **Remote Docker Context**: `docker context create dev1 --docker "host=ssh://dev1"`
   - Chạy Docker trên dev1, build/pull image từ MacBook
   - Hoạt động tốt, giải phóng 15GB trên Mac ngay lập tức

2. **VS Code Remote SSH**: Connect VS Code vào dev1, edit code trực tiếp
   - Chạy được, nhưng VS Code remote hay lag, extension hay lỗi

3. **rsync**: Sync code từ Mac lên dev1, chạy service trên dev1
   - `rsync -avz --exclude=vendor ./project dev1:~/projects/`
   - Nhanh, đơn giản, nhưng phải nhớ sync manual

**Kết quả**: Hoạt động! Nhưng phải nhớ quá nhiều lệnh, config thủ công từng project,
không có cách nào reproduce cho người khác trong team.

---

## Vấn đề cần giải quyết

### Cho individual developer
- Laptop hết dung lượng vì Docker, project files, cache
- Có server/desktop rảnh nhưng không biết tận dụng hiệu quả
- Remote dev setup thủ công, phải nhớ nhiều lệnh, dễ quên config

### Cho team nhỏ (2-20 người)
- **Onboarding mất 2-3 ngày**: clone repo, cài Docker, setup database, config env,
  debug "nó chạy trên máy anh mà"
- Senior phải ngồi hỗ trợ junior setup môi trường thay vì làm feature
- Mỗi dev config khác nhau → bug "works on my machine"
- Không có DevOps → không ai maintain dev environment

### Cho AI coding workflow
- Claude Code / AI agent cần chạy trên máy có compute
- Laptop fan quay vù vù khi AI agent build + test
- Muốn chạy nhiều agent song song nhưng laptop chỉ có giới hạn resource
- Agent conflict khi share cùng filesystem, database, port

---

## Nghiên cứu giải pháp hiện có

### DevPod (Loft Labs) — 15k+ GitHub stars

**Ý tưởng rất tốt**: open-source, client-only, devcontainer spec, multi-provider.

**Nhưng đang chết dần**:
- Release cuối cùng: **June 2024** — gần 1 năm không update
- Bug nghiêm trọng chưa fix:
  - Giới hạn 4 SSH session → không dùng được với nhiều terminal
  - JetBrains integration hay bị hang
  - CRLF bug trên Windows
- Community đang mất niềm tin, issues chất đống không ai xử lý
- Loft Labs có vẻ chuyển focus sang vCluster (sản phẩm enterprise)

**Bài học**: Ý tưởng đúng, execution không sustain được. devbox học từ DevPod:
giữ lại ý tưởng client-only + devcontainer, nhưng đơn giản hơn, focus hơn.

### GitHub Codespaces

**Pros**: Tích hợp sâu GitHub, devcontainer native, AI Copilot tích hợp.

**Cons**:
- **Giá**: $0.18/giờ cho 4-core → ~$40-80/dev/tháng nếu dùng full-time
- Team 10 người = $400-800/tháng chỉ cho dev environment
- Với startup bootstrap, đó là tiền thuê thêm 1 người
- Data nằm trên cloud Microsoft → compliance concern
- Latency phụ thuộc internet quality

### Coder

**Pros**: Self-hosted, mạnh, enterprise-grade, Terraform provider.

**Cons**:
- **Quá phức tạp** cho team nhỏ: cần Kubernetes hoặc VM provisioner
- Setup ban đầu mất 1-2 ngày cho người có kinh nghiệm
- Maintain cần DevOps knowledge
- Overkill cho team 5-10 người với 1-2 server

### Gitpod (Classic)

- **Đã shutdown** cloud offering
- Open-source version phức tạp, cần Kubernetes
- Không còn là option viable

### Zed Editor — Remote Development

**Phát hiện quan trọng**: Zed đang phát triển remote development architecture
rất tốt:
- **Performance vượt trội**: Native code (Rust), khởi động < 1 giây
- **Claude Code tích hợp native** qua ACP (Agent Communication Protocol)
- **Remote architecture**: Zed server chạy trên remote, client render UI local
- **Collab native**: Pair programming real-time built-in

Zed là editor phù hợp nhất cho remote dev workflow vì nó được thiết kế
remote-first, không phải bolt-on như VS Code Remote.

### Kết luận nghiên cứu

| Tool | Self-hosted | Simple | Cheap | Maintained |
|------|:---------:|:------:|:-----:|:---------:|
| Codespaces | No | Yes | No | Yes |
| Coder | Yes | No | Yes | Yes |
| DevPod | Yes | ~Yes | Yes | **No** |
| Gitpod | Yes | No | Yes | **No** |
| **devbox** | **Yes** | **Yes** | **Yes** | **Yes** |

**Không có tool nào simple + self-hosted + lightweight cho team nhỏ (2-20 người)
với hardware có sẵn.** Đó là khoảng trống mà devbox nhắm vào.

---

## Product Vision

### Một câu

> "Biến bất kỳ máy Linux nào thành dev environment sẵn sàng dùng trong 1 lệnh."

### Positioning

**Simple + Self-hosted + Cheap.**

- Codespaces: managed nhưng **đắt** ($40-80/dev/tháng)
- Coder: self-hosted nhưng **phức tạp** (cần Kubernetes)
- DevPod: simple nhưng **đang chết** (không maintain)
- Gitpod Classic: **đã shutdown**
- **devbox**: simple + self-hosted + cheap + alive

### Core promise

Bạn có 1 con máy Linux (desktop cũ, mini PC, VPS rẻ) → cài devbox →
team member chạy `devbox up` → có dev environment đầy đủ trong 5 phút.

Không cần:
- Cloud account
- Kubernetes
- DevOps engineer
- Đọc 50 trang docs

---

## Công dụng lớn nhất

### Xoá khoảng cách giữa "vào team" và "bắt đầu code"

**Trước devbox:**
1. Clone repo (10 phút — repo lớn)
2. Cài Docker Desktop (15 phút — download + restart)
3. Copy `.env.example` → `.env`, sửa config (10 phút)
4. `docker compose up` → lỗi port conflict (15 phút debug)
5. Database migration → lỗi version (30 phút hỏi senior)
6. Seed data → lỗi memory (20 phút tăng Docker memory)
7. Chạy app → lỗi SSL cert local (30 phút setup mkcert)
8. **Tổng: 2-3 ngày** (tính cả thời gian chờ senior support)

**Sau devbox:**
1. `devbox up project-name`
2. Đợi 5 phút (pull image, clone repo, run migration, seed)
3. Mở Zed/VS Code → connect → code ngay
4. **Tổng: 5 phút**

### Giá trị kinh tế

- Senior developer: ~$50/giờ
- 2-3 ngày hỗ trợ onboard = $800-1200 lãng phí
- Team tuyển 5 người/năm = $4000-6000/năm chỉ cho onboarding
- devbox giảm xuống gần 0

---

## Multi-agent collaboration (killer feature)

### Vấn đề hiện tại

Khi dùng AI coding agent (Claude Code, Cursor, Copilot Workspace):
- Agent chạy trên laptop → fan quay, máy nóng, battery hết
- 1 agent = 1 workspace = 1 branch → sequential, chậm
- 2 agent trên cùng 1 máy → conflict filesystem, port, database

### Giải pháp: Agent Farm

Mỗi AI agent nhận **1 workspace riêng biệt** trên server:

```
Server (dev1: i5, 16GB RAM)
├── workspace-1 (Agent A): feature/auth
│   ├── filesystem riêng (/workspaces/auth/)
│   ├── database riêng (MySQL port 13306)
│   ├── app port riêng (18080)
│   └── git branch riêng (feature/auth)
├── workspace-2 (Agent B): feature/payment
│   ├── filesystem riêng (/workspaces/payment/)
│   ├── database riêng (MySQL port 23306)
│   ├── app port riêng (28080)
│   └── git branch riêng (feature/payment)
└── workspace-3 (Agent C): fix/bug-123
    ├── filesystem riêng (/workspaces/bug-123/)
    ├── database riêng (MySQL port 33306)
    ├── app port riêng (38080)
    └── git branch riêng (fix/bug-123)
```

**3 agent chạy song song = 3x tốc độ phát triển**

### Workflow

```
Developer trên MacBook:
1. devbox up auth --branch feature/auth         # tạo workspace cho auth
2. devbox up payment --branch feature/payment   # tạo workspace cho payment
3. devbox up bug-123 --branch fix/bug-123       # tạo workspace cho bug fix

# Mỗi workspace tự động:
# - Clone repo + checkout branch
# - Chạy docker compose (app + db + cache)
# - Expose port qua Tailscale
# - Sẵn sàng cho AI agent connect

4. Claude Code agent-1 → connect workspace-1 → code auth feature
5. Claude Code agent-2 → connect workspace-2 → code payment
6. Claude Code agent-3 → connect workspace-3 → fix bug

# Developer review tất cả kết quả, merge PRs
```

### Scale

- 1 server i5/16GB → chạy được 3-5 workspace cùng lúc
- Cần thêm? Thêm server, devbox tự phân bổ workspace across servers
- Server thành **"agent farm"** — mỗi agent có sandbox riêng, không conflict
- Cost: ~$200-500 cho 1 mini PC (NUC, Beelink) vs $80/tháng cho Codespaces

---

## Target Users

### Primary: Dev team nhỏ (2-20 người)

- Bootstrap / early-stage startup
- Không có DevOps dedicated
- Dùng Docker cho local dev
- Muốn giảm thời gian onboarding
- Có ít nhất 1 server Linux (desktop cũ, NUC, hoặc VPS rẻ)

**Persona**: CTO/Lead dev ở startup 8 người, dùng Laravel/Rails/Next.js,
team toàn junior-mid, mỗi lần có người mới vào mất 2 ngày setup.

### Secondary: Individual developer

- Laptop SSD nhỏ (256GB), hết dung lượng
- Có máy desktop/server ở nhà hoặc VPS
- Muốn offload Docker + heavy build sang máy khác
- Dùng AI coding agent, muốn chạy trên server mạnh hơn

**Persona**: Developer dùng MacBook Air M2 256GB, có con Ubuntu desktop
ở nhà, muốn dev trên đó thay vì chật chội trên laptop.

---

## Tại sao bây giờ

### 1. Laptop ngày càng mỏng, SSD không nâng được
- Apple solder SSD vào board từ 2016
- Ultrabook Windows cũng đang theo trend này
- 256GB là config phổ biến nhất (giá rẻ nhất)
- Dev environment ngày càng nặng (Docker, node_modules, AI models)

### 2. Tailscale/WireGuard đã giải quyết network layer
- Trước 2020: VPN phức tạp, NAT traversal khó
- Tailscale: cài 1 lệnh → mesh network → mọi máy connect được
- Latency ~5ms trên cùng LAN, ~30ms cross-region
- **Network không còn là bottleneck cho remote dev**

### 3. Zed đang mature, remote-first architecture
- Zed: native performance, Rust, remote dev built-in
- Claude Code tích hợp native qua ACP
- Không cần extension ecosystem như VS Code → ít bug hơn
- Collab native → pair programming real-time

### 4. Khoảng trống thị trường
- DevPod: release cuối June 2024, community mất niềm tin
- Gitpod Classic: đã shutdown
- Coder: vẫn active nhưng quá phức tạp cho team nhỏ
- Codespaces: vẫn đắt
- **Không ai phục vụ team nhỏ self-hosted**

### 5. AI coding agents cần compute
- Claude Code, Cursor, Copilot chạy heavy workload
- Build + test + lint trên laptop → máy nóng, pin hết
- Server có nhiều CPU/RAM hơn → agent chạy nhanh hơn
- Nhiều agent song song → cần nhiều isolated workspace

### 6. devcontainer là chuẩn OCI
- devcontainer.json được VS Code, Codespaces, DevPod support
- Không cần invent format mới
- User đã có devcontainer.json → migrate sang devbox dễ dàng
- Cộng đồng devcontainer features đang grow

---

## Architecture

### High-level

```
┌─────────────────────────────────────────────────────┐
│                    Developer Machine                 │
│                    (MacBook, thin client)            │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌────────────────────┐ │
│  │ Zed      │  │ Terminal  │  │ devbox CLI         │ │
│  │ (editor) │  │ (ssh)    │  │ (workspace mgmt)   │ │
│  └────┬─────┘  └────┬─────┘  └────────┬───────────┘ │
│       │              │                 │              │
└───────┼──────────────┼─────────────────┼─────────────┘
        │              │                 │
        └──────────────┼─────────────────┘
                       │
              ┌────────▼────────┐
              │   Tailscale     │
              │   (network)     │
              └────────┬────────┘
                       │
┌──────────────────────┼──────────────────────────────┐
│                Server (dev1, dev2, VPS...)           │
│                      │                               │
│  ┌───────────────────▼─────────────────────────┐    │
│  │              devbox agent (future)           │    │
│  │              hoặc SSH + Docker              │    │
│  └───────────┬──────────────┬──────────────┐    │    │
│              │              │              │    │    │
│  ┌───────────▼──┐ ┌────────▼───┐ ┌────────▼──┐│    │
│  │ Workspace 1  │ │ Workspace 2│ │ Workspace 3││    │
│  │ (container)  │ │ (container)│ │ (container)││    │
│  │              │ │            │ │            ││    │
│  │ app:8080     │ │ app:8081   │ │ app:8082  ││    │
│  │ db:3306      │ │ db:3307    │ │ db:3308   ││    │
│  │ redis:6379   │ │ redis:6380 │ │ redis:6381││    │
│  └──────────────┘ └────────────┘ └────────────┘│    │
│                                                 │    │
└─────────────────────────────────────────────────────┘
```

### Stack layers

```
Layer 4: Editor        → Zed (primary), VS Code (compatible)
Layer 3: devbox        → Workspace lifecycle management
Layer 2: Container     → Docker + devcontainer.json spec
Layer 1: Network       → Tailscale (mesh VPN, DNS, ACL)
Layer 0: Hardware      → Any Linux machine (desktop, NUC, VPS)
```

### Data flow: `devbox up project-name`

```
1. devbox CLI đọc devbox.yaml (hoặc devcontainer.json) của project
2. SSH vào server target (dev1)
3. Clone repo (hoặc sync code) vào /workspaces/{name}/
4. Tạo Docker network isolated cho workspace
5. Chạy docker compose up (app + services)
6. Chạy database migration + seed
7. Expose port qua Tailscale serve (HTTPS tự động)
8. Output: URL workspace, SSH command, Zed connect command
```

### Networking model

Tailscale giải quyết mọi vấn đề network:
- **DNS**: `workspace-name.dev1.tail1234.ts.net` → truy cập từ bất kỳ đâu
- **HTTPS**: Tailscale tự động cấp cert → không cần mkcert
- **ACL**: Giới hạn ai truy cập workspace nào
- **NAT traversal**: Hoạt động qua firewall, không cần port forwarding

---

## Tech Decisions

### Language: Go

**Lý do chọn Go**:
- **Single binary**: Distribute dễ, không cần runtime
- **Performance**: Nhanh hơn Python/Node cho CLI tool
- **Ecosystem**: Docker SDK, Tailscale SDK đều viết bằng Go
- **DevPod cũng dùng Go**: Community quen thuộc, có thể reuse concepts
- **Cross-compile**: Build cho Linux/Mac/Windows từ 1 codebase

### Network: Tailscale

**Lý do chọn Tailscale**:
- Cài 1 lệnh, hoạt động ngay
- Mesh network: mọi máy connect trực tiếp, không qua central server
- `tailscale serve`: expose local port ra HTTPS với cert tự động
- DNS tự động: `hostname.tailnet.ts.net`
- ACL: kiểm soát access chi tiết
- Free tier đủ dùng cho team nhỏ (100 devices)
- SDK/CLI integration tốt

### Container: Docker + devcontainer.json

**Lý do**:
- Docker là standard, mọi dev đã biết
- devcontainer.json là chuẩn OCI, nhiều tool support
- Không cần invent container format mới
- devcontainer features: thêm tool vào container dễ dàng
- Docker Compose cho multi-service (app + db + cache)

### Editor: Zed-first, VS Code compatible

**Lý do chọn Zed primary**:
- Performance native (Rust), khởi động < 1s
- Remote dev architecture tốt hơn VS Code
- Claude Code tích hợp native qua ACP
- Collab built-in
- Đang grow nhanh, community active

**VS Code vẫn support** vì:
- Nhiều dev đang dùng VS Code
- VS Code Remote SSH hoạt động được
- Migration path: bắt đầu với VS Code, chuyển sang Zed sau

### License: MPL-2.0

**Lý do chọn MPL-2.0**:
- Copyleft ở mức file (không phải toàn project như GPL)
- Cho phép sử dụng commercial
- Bắt buộc share changes cho file MPL
- Cho phép kết hợp với code proprietary
- HashiCorp (Terraform, Vagrant) cũng dùng MPL → precedent tốt

---

## Roadmap

### Phase 0: Dogfood (1-2 tuần)

**Mục tiêu**: Team mình dùng hàng ngày, validate workflow.

Scope:
- CLI commands: `devbox up`, `devbox stop`, `devbox list`, `devbox destroy`, `devbox ssh`
- Config: `devbox.yaml` per-project
- Backend: Docker Compose trên server qua SSH
- Network: Tailscale serve cho mỗi workspace
- Editor: Zed remote connect

**Không làm**:
- devcontainer.json support (dùng devbox.yaml riêng trước)
- Multi-server (chỉ 1 server dev1)
- Web UI
- Authentication/authorization

### Phase 1: MVP (4-6 tuần)

**Mục tiêu**: Người ngoài team cài được, dùng được.

Scope:
- devcontainer.json support (đọc config từ `.devcontainer/devcontainer.json`)
- `devbox init` — tạo config cho project mới
- `devbox doctor` — check prerequisites (Docker, Tailscale, SSH)
- Documentation: README, quick start, troubleshooting
- GitHub releases với binary cho Linux/Mac
- Basic error handling + logging

### Phase 2: Multi-user (4-6 tuần)

**Mục tiêu**: Nhiều dev share 1 server, workspace isolation.

Scope:
- User management: ai có quyền tạo workspace trên server nào
- Resource limits: CPU/RAM per workspace (cgroups hoặc Docker limits)
- Workspace naming: `{user}-{project}-{branch}`
- Port allocation: tự động assign port không conflict
- `devbox server add/remove` — quản lý server pool
- Multi-server: devbox chọn server có resource trống

### Phase 3: TUI/Web UI (4-6 tuần)

**Mục tiêu**: Dashboard quản lý workspace, không cần nhớ lệnh.

Scope:
- TUI (terminal UI) dùng Bubble Tea / Charm
- Web dashboard (optional): list workspace, start/stop, logs
- Workspace templates: tạo workspace từ template project
- Snapshot/restore: save workspace state, restore later
- Metrics: CPU/RAM/disk usage per workspace

### Phase 4: Community (ongoing)

**Mục tiêu**: Open source, community adoption.

Scope:
- Plugin system: custom provider (Docker, Podman, LXC)
- Community templates: share workspace templates
- CI/CD integration: tạo workspace cho PR preview
- Marketplace: devcontainer features + devbox plugins
- Enterprise features (nếu có demand): SSO, audit log, compliance

---

## Nguyên tắc thiết kế

### 1. Simple by default, powerful when needed
- `devbox up` phải hoạt động với minimal config
- Advanced features qua flags/config, không bắt buộc
- README 5 phút đọc xong, bắt đầu dùng được

### 2. Convention over configuration
- Default server: đầu tiên trong server list
- Default port: tự động allocate
- Default branch: current branch hoặc main
- Chỉ cần config khi muốn khác default

### 3. Fail fast, fix fast
- Error messages rõ ràng, gợi ý cách fix
- `devbox doctor` check mọi prerequisite
- Logs verbose khi cần debug

### 4. Respect existing tools
- Dùng Docker, không reinvent container runtime
- Dùng Tailscale, không reinvent networking
- Dùng devcontainer spec, không reinvent config format
- Dùng SSH, không reinvent remote access

### 5. Offline-first thinking
- devbox CLI không cần internet (chỉ cần Tailscale network)
- Không có SaaS dependency
- Không telemetry, không tracking
- Data ở server của bạn, 100% self-hosted

---

## Metrics thành công

### Phase 0 (Dogfood)
- Team mình dùng hàng ngày mà không quay lại local Docker
- `devbox up` < 5 phút cho project mới
- Onboarding người mới: < 30 phút (bao gồm cài devbox + tạo workspace đầu tiên)

### Phase 1 (MVP)
- 10+ GitHub stars (có người ngoài team biết đến)
- 3+ người ngoài team cài được và dùng được (không cần hỗ trợ)
- README → dùng được trong 15 phút

### Phase 2 (Multi-user)
- 3+ dev trong team dùng chung 1 server mà không conflict
- 100+ GitHub stars

### Phase 3+
- 500+ GitHub stars
- Community contributions (PRs, plugins, templates)
- Nhắc đến trong "awesome-remote-dev" lists

---

*Document này sẽ được update khi có thêm insights từ quá trình dogfood và feedback.*

*Tạo: 2026-04-10*
*Cập nhật lần cuối: 2026-04-10*
