# Demo 0001: Basic Setup and Content Delivery

## Overview

This demo showcases the core functionality of Wrale Signage:
- Display registration using human-readable codes
- Content delivery and sequencing
- Health monitoring and error recovery

## Prerequisites

1. PostgreSQL 14+
2. Go 1.21+
3. Node.js 18+
4. Docker (for development database)

## Setup Steps

1. Start development database:
```bash
docker-compose up -d postgres
./scripts/init-test-db.sh
```

2. Build and start the server:
```bash
make build
./bin/wsignd --config configs/dev.yaml
```

3. Build and start the web interface:
```bash
cd web
npm install
npm run dev
```

## Demo Flow

### 1. Display Registration (5 minutes)

1. Open web interface at http://localhost:3000
2. Browser interface loads and displays registration code (e.g., "BLUE-FISH")
3. Use CLI to register display:
```bash
./bin/wsignctl display activate BLUE-FISH \
  --site headquarters \
  --zone lobby \
  --position main
```
4. Observe display transitioning to registered state

### 2. Content Management (5 minutes)

1. Add test content:
```bash
./bin/wsignctl content add \
  --path testdata/welcome.html \
  --duration 10s
  
./bin/wsignctl content add \
  --path testdata/news.html \
  --duration 15s
```

2. Create content sequence:
```bash
./bin/wsignctl rule add \
  --display BLUE-FISH \
  --content welcome.html,news.html
```

3. Observe content rotation in browser interface

### 3. Health Monitoring (5 minutes)

1. View current display status:
```bash
./bin/wsignctl display list
```

2. Check content health metrics:
```bash
./bin/wsignctl content health welcome.html
```

3. Simulate error conditions:
   - Stop web server serving news.html
   - Observe automatic fallback to welcome.html
   - Watch error reporting in logs

## Key Points to Demonstrate

1. Human-Readable Setup
   - No complex configuration required
   - Display setup uses simple codes
   - CLI provides intuitive management

2. Resilient Operation  
   - Content caching for offline operation
   - Automatic error recovery
   - Graceful fallback behavior

3. Monitoring Capabilities
   - Real-time health metrics
   - Content performance tracking
   - Error detection and reporting

## Cleanup

1. Stop all components:
```bash
docker-compose down
```

2. Reset database:
```bash
./scripts/init-test-db.sh --reset
```

## Additional Notes

- Demo uses development configuration
- Content paths are relative to configs/content directory
- Health metrics persist in database between runs
- Use --verbose flag with CLI for detailed output