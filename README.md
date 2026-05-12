# 👛 Pocka: The Viral Telegram Expense Tracker

Pocka is a production-grade, highly scalable Telegram bot designed to help users track their daily expenses effortlessly. Built with **Go** and **Ent ORM**, it focuses on simplicity, speed, and viral growth features like streaks and referrals.

---

## ✨ Features

- **🚀 Natural Parsing**: Log expenses by simply typing (e.g., `100 lunch`, `50.50 taxi`).
- **📊 Weekly Stats**: Get a breakdown of your spending habits with the `/stats` command.
- **🔥 Streak System**: Stay motivated by maintaining a daily logging streak.
- **🌍 Multi-Country Ready**: Built-in support for different currencies and timezones.
- **🔗 Referral Engine**: Built-in referral tracking to drive organic growth.
- **🏗️ Clean Architecture**: Interface-driven design, ready for AI-powered parsing upgrades.

---

## 🛠️ Tech Stack

- **Language**: [Go (1.26+)](https://golang.org/)
- **ORM**: [Ent](https://entgo.io/) (Schema-first, type-safe ORM)
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) (For webhooks)
- **Database**: PostgreSQL
- **Bot API**: [Telegram Bot API v5](https://github.com/go-telegram-bot-api/telegram-bot-api)
- **Infrastructure**: Docker & Docker Compose

---

## 🚀 Quick Start

### 1. Prerequisites
- Go 1.26+
- Docker & Docker Compose
- A Telegram Bot Token from [@BotFather](https://t.me/botfather)

### 2. Installation
Clone the repository and install dependencies:
```bash
git clone https://github.com/your-username/pocka.git
cd pocka
go mod tidy
```

### 3. Configuration
Create a `.env` file in the root:
```bash
TELEGRAM_TOKEN=your_bot_token_here
DATABASE_URL=postgresql://pocka_user:pocka_pass@localhost:5432/pocka_db?sslmode=disable
PORT=8080
ENVIRONMENT=development
# WEBHOOK_URL=https://your-domain.com (Leave empty for long-polling)
```

### 4. Run with Docker
Start the database and the application:
```bash
docker-compose up -d
```

Or run the Go application directly (after starting the DB):
```bash
go run cmd/bot/main.go
```

---

## 🤖 Bot Commands

- `/start` - Initialize your account and get a referral link.
- `/stats` - View your weekly spending breakdown.
- `/help` - Show usage instructions.
- `[amount] [category]` - Just type your expense to log it!

---

## 🏗️ Project Structure

```text
pocka/
├── cmd/bot/                # Entry point (main.go)
├── internal/
│   ├── bot/                # Telegram handlers & webhook logic
│   ├── core/               # Domain interfaces (Ports)
│   ├── services/           # Business logic (Expenses, Streaks, Stats)
│   ├── parsers/            # Regex-based expense parser
│   └── storage/            # Database initialization
├── ent/                    # Generated Ent ORM code & schemas
└── docker-compose.yml      # Local dev environment
```

---

## 🛣️ Roadmap

- [ ] **AI Parser**: Integration with LLMs (Gemini/OpenAI) for complex sentence parsing.
- [ ] **Image Generation**: Automated "Milestone Cards" for social sharing.
- [ ] **Budgeting**: Set monthly limits and receive notifications.
- [ ] **Export**: Export expenses to CSV/Excel.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
