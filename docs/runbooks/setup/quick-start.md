# Quick Start Guide

**Get llm-viz running in 2 minutes with Docker Compose**

---

## Prerequisites

**Option A: Docker Compose** (Recommended)
- Docker Desktop 4.0+
- Anthropic or OpenAI API key

**Option B: Manual Setup**
- Node.js 18+
- Go 1.23+
- Anthropic or OpenAI API key

---

## 🚀 Quick Start with Docker Compose (2min)

### Step 1: Clone & Configure (1min)

```bash
# Clone repository
git clone https://github.com/coolexplorer/llm-viz.git
cd llm-viz

# Configure API keys
cp .env.example .env
nano .env  # Add your API keys
```

Example `.env`:

```bash
ANTHROPIC_API_KEY=sk-ant-api03-xxxxx
OPENAI_API_KEY=sk-xxxxx  # Optional
```

### Step 2: Start Services (1min)

```bash
# Build and start all services
docker-compose up --build

# Or run in background
docker-compose up -d --build
```

**✅ That's it!** Services are now running:
- Backend: http://localhost:8080
- Frontend: http://localhost:3000

### Stop Services

```bash
# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

---

## 🛠️ Manual Setup (Alternative Method)

### Step 1: Clone & Install (2min)

```bash
# Clone repository
git clone https://github.com/coolexplorer/llm-viz.git
cd llm-viz

# Backend
cd backend
go mod download

# Frontend (new terminal)
cd frontend
npm install
```

### Step 2: Configure API Keys (1min)

```bash
# Backend
cd backend
cp .env.example .env
nano .env  # Add your API keys
```

Example `.env`:

```bash
ANTHROPIC_API_KEY=sk-ant-api03-xxxxx
OPENAI_API_KEY=sk-xxxxx  # Optional
PORT=8080
```

### Step 3: Start Services (2min)

**Terminal 1 - Backend:**

```bash
cd backend
go run ./cmd/server
```

**Terminal 2 - Frontend:**

```bash
cd frontend
npm run dev
```

---

## Step 4: Test (30sec)

1. Open http://localhost:3000
2. Select provider (Anthropic or OpenAI)
3. Choose model
4. Type a message
5. Watch tokens/cost update in real-time

---

## ✅ Success Checklist

- [ ] Backend running on :8080
- [ ] Frontend running on :3000
- [ ] Dashboard loads in browser
- [ ] Provider selector shows your providers
- [ ] Test message returns response
- [ ] Token counter updates
- [ ] Cost tracker shows USD

---

## Next Steps

- [Claude Code Integration](../integration/claude-code-integration.md) - Connect to your AI workflow
- [Environment Configuration](environment-config.md) - Advanced settings
- [Production Setup](production-setup.md) - Deploy to cloud

---

## Troubleshooting

**Issue**: Backend fails to start

```bash
# Check Go version
go version  # Should be 1.23+

# Check API key format
echo $ANTHROPIC_API_KEY | grep "sk-ant-"
```

**Issue**: Frontend won't load

```bash
# Check Node version
node --version  # Should be 18+

# Clear cache
rm -rf .next node_modules
npm install
npm run dev
```

---

**Need help?** See [Common Issues](../troubleshooting/common-issues.md)
