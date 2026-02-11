# lx — Real-time Log Monitoring & Extraction Tool

`lx` is a lightweight, high-performance CLI tool designed for real-time log monitoring, filtering, and extraction.  
It combines the power of `grep`, `tail`, and `awk` into a modern, developer-friendly interface with built-in TUI, Docker support, and structured log parsing.

![Version](https://img.shields.io/badge/version-0.2.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Go](https://img.shields.io/badge/go-1.21+-00ADD8)

## ✨ Key Features

- **Pipeline Architecture**: Modular design for Source → Filter → Sink processing.
- **TUI Dashboard**: Interactive terminal UI with real-time viewport, scroll, search, and rate visualization.
- **Smart Filtering**:
  - Regex & Keyword support (AND/OR modes)
  - Log Level auto-detection & filtering
  - Context awareness (`--before`, `--after`)
  - Noise reduction via `--exclude`
- **Multi-Source**:
  - `stdin` pipe support
  - File following (`tail -f` style)
  - **Docker** container log streaming (`--docker`)
- **Structured Parsing**: Built-in **Grok** parser for extracting fields from unstructured logs.

## 📦 Installation

```bash
go install github.com/Geun-Oh/lx@latest
```

## 🚀 Usage

### basic

```bash
# Execute command and filter logs
lx -k ERROR -- ./my-app

# Pipe from other tools
kubectl logs -f pod-name | lx -k ERROR
```

### Modes & Flags

#### 1. Input Sources

| Flag           | Description                | Example                  |
| -------------- | -------------------------- | ------------------------ |
| `--file, -f`   | Read from file             | `lx -f /var/log/syslog`  |
| `--follow`     | Follow file (like tail -f) | `lx -f app.log --follow` |
| `--docker, -d` | Stream logs from container | `lx -d my-container`     |
| `stdin`        | Pipe input                 | `cat file.log \| lx`     |

#### 2. Filtering

| Flag            | Description                      | Example                         |
| --------------- | -------------------------------- | ------------------------------- |
| `--keyword, -k` | Filter by substring (repeatable) | `lx -k "timeout" -k "refused"`  |
| `--regex, -r`   | Filter by regex pattern          | `lx -r "status=5\d{2}"`         |
| `--level, -l`   | Filter by log severity           | `lx -l ERROR,WARN`              |
| `--exclude, -e` | Exclude matching lines           | `lx -e "healthcheck"`           |
| `--match-mode`  | Combine filters (`and`/`or`)     | `lx -k A -k B --match-mode and` |
| `--before, -B`  | Print N lines before match       | `lx -k ERROR -B 5`              |
| `--after, -A`   | Print N lines after match        | `lx -k ERROR -A 5`              |

#### 3. TUI & Monitoring

| Flag           | Description                      | Example                      |
| -------------- | -------------------------------- | ---------------------------- |
| `--tui`        | Launch interactive dashboard     | `lx --tui -k ERROR -- ./app` |
| `--alert`      | Alert on regex match (TUI flash) | `lx --tui --alert "panic"`   |
| `--alert-rate` | Alert on rate spike (lines/s)    | `lx --tui --alert-rate 100`  |
| `--stats`      | Show summary on exit             | `lx --stats -- ./app`        |

#### 4. Output & Parsing

| Flag           | Description                    | Example                    |
| -------------- | ------------------------------ | -------------------------- |
| `--format`     | Output format (`text`, `json`) | `lx --format json`         |
| `--color`      | Colorize output by level       | `lx --color`               |
| `--output, -o` | Write to file                  | `lx -o filtered.log`       |
| `--grok`       | Parse fields using Grok        | `lx --grok "%{IP:client}"` |

### TUI keybindings

- `/`: Search (type query, `Enter` to jump, `Esc` to cancel)
- `p`: Pause/Resume auto-scroll
- `g` / `G`: Jump to bottom / top
- `↑` / `↓` : Scroll manualy
- `q`: Quit

## 💡 Examples

**Monitor Docker logs for errors with TUI:**

```bash
lx --docker web-server --tui --level ERROR,WARN --alert "panic"
```

**Extract structured data from access logs:**

```bash
lx -f access.log --grok "%{IP:client} %{WORD:method} %{NUMBER:duration}" --format json
```

**Debug a crashing app with context:**

```bash
./my-app | lx -k "panic" -B 10 -A 5 --color
```

## 🤝 Contribution

Contributions are welcome! Please follow these steps:

1. Fork the project.
2. Create your feature branch (`git checkout -b feature/new-feature`).
3. Commit your changes (`git commit -m 'Add some new-feature'`).
4. Push to the branch (`git push origin feature/new-feature`).
5. Open a Pull Request.

## 📄 License

Distributed under the **MIT License**. See `LICENSE` for more information.
