# 👑 Monarch

### The Sovereign System Orchestrator

**"Don't just manage your OS. Rule it."**

Monarch is a next-generation orchestration tool that transforms Linux system management from a complex and fragile process into a modular, reversible, and declarative **Lego experience**.

[Vision](https://www.google.com/search?q=%23-vision-invisible-os "null") • [How It Works](https://www.google.com/search?q=%23-key-features "null") • [Comparison](https://www.google.com/search?q=%23-why-monarch "null") • [Roadmap](https://www.google.com/search?q=%23-roadmap "null")

## 🔮 Vision: "Invisible OS"

Setting up and maintaining a modern Linux environment (e.g., CachyOS + Hyprland) is chaotic. Dotfiles, packages, systemd services, and user permissions are disconnected. Change one thing, and the system gets "dirty"; undoing changes is nearly impossible.

**Monarch ends this chaos.**

It treats the system not as a monolithic entity, but as a collection of attachable and detachable **Rulesets**.

- **Attach:** Apply the "Gaming Mode" ruleset. (Steam installs, drivers configure, kernel optimizes.)
    
- **Detach:** Done gaming? Remove the ruleset. Monarch deletes the packages, reverts the settings, and cleans up generated files, leaving the system **pristine**.
    
- **Defend (Self-Heal):** With the background Sentinel, if a file is manually corrupted, Monarch instantly repairs it.
    

## 🚀 Key Features

### 1. Declarative & State-Aware

Monarch doesn't run commands blindly. It first analyzes the system's `Current State`, compares it with your `Desired State`, and applies only the necessary `Diff`.

### 2. The Lego Principle (Atomic Rulesets)

An application is never just a "package". For Monarch, a _Ruleset_ is an atomic unit containing the package, configuration files, service definitions, and necessary user permissions.

### 3. Agentless Architecture

No need for Python, Ruby, or an installed agent on the target machine. Monarch is written in **Go** and runs as a single binary. It temporarily copies itself via SSH, executes the task, and vanishes without a trace.

### 4. Sovereignty

Manage your entire fleet—from your personal laptop to remote VPS servers—from a single control center.

## 🆚 Why Monarch?

Monarch combines the power of Ansible, the determinism of NixOS, and the state management of Terraform into a user-friendly package.

|   |   |   |   |   |
|---|---|---|---|---|
|**Feature**|**👑 Monarch**|**🐍 Ansible**|**❄️ NixOS**|**🐚 Shell Scripts**|
|**Language / Speed**|**Go (Compiled, Blazing Fast)**|Python (Slow)|Nix (Complex)|Bash (Fast but unsafe)|
|**Undo / Revert**|✅ **Native (Automatic)**|❌ None (Manual)|✅ (Rollback)|❌ None|
|**State Tracking**|✅ **State.json + Checksum**|❌ Limited (Facts)|✅ (Store)|❌ None|
|**Dependencies**|**None (Single Binary)**|Requires Python|Requires Specific OS|Dependency Hell|
|**Learning Curve**|**Low (Lego Logic)**|Medium (YAML clutter)|Very High|Variable|
|**Use Case**|Desktop & Server|Server Focused|Entire OS|Simple Tasks|

## 🏗️ Architecture: The Holy Trinity

The Monarch ecosystem is built on three main pillars:

1. **Monarch Engine (CLI):** The brain. A Go-based core handling `resource`, `apply`, and `diff` logic.
    
2. **Monarch Hub (The Library):** GitHub-based global ruleset library. Pull "Hyprland Setup" or "DevOps Stack" rulesets created by others with a single command.
    
3. **Monarch Studio (GUI):** A modern desktop interface built with Wails to conquer terminal fear. Manage your system like a cockpit.
    

## 🛠️ Tech Stack

- **Core:** [Go (Golang)](https://go.dev/ "null") - High performance and concurrency.
    
- **Config:** YAML - Human-readable, simple structure.
    
- **State:** JSON - Portable and lightweight state tracking.
    
- **Security:** [Age (X25519)](https://github.com/FiloSottile/age "null") - Modern and secure secret management.
    
- **Transport:** SSH - Secure remote server management.
    

## ⚡ Quick Start (Alpha)

Monarch is currently in active development. To try it out:

```
# 1. Clone the repository
git clone [https://github.com/melih-ucgun/monarch.git](https://github.com/melih-ucgun/monarch.git)
cd monarch

# 2. Build
go build -o monarch main.go

# 3. Apply an example configuration (Dry-Run)
./monarch apply --config monarch.yaml --dry-run
```

### Example `monarch.yaml`

```
resources:
  - type: package
    id: neovim
    name: neovim
    state: present

  - type: file
    id: nvim-config
    path: ~/.config/nvim/init.lua
    content: |
      print("Hello from Monarch Managed Config!")
    owner: melih
    mode: "0644"

  - type: service
    id: docker-service
    name: docker
    state: running
    enabled: true
```

## 🗺️ Roadmap

Monarch is constantly evolving. Here is our plan:

- [x] **Core (Ready):** Basic commands, file/package management, and state tracking.
    
- [ ] **Undo & Hub (Next):** `Undo` feature and GitHub integration.
    
- [ ] **Interface (GUI):** Modern desktop application and Hyprland integration.
    
- [ ] **Autonomous:** Self-healing system and fleet management.
    

**Monarch** © 2025 Developed by Melih Uçgun
