# Demo 0001: Basic Setup and Content Delivery

## Overview

This demo showcases the core functionality of Wrale Signage:
- Display registration using human-readable codes
- Content delivery and sequencing via distinct content URLs
- Health monitoring and error recovery

## Prerequisites

1. Docker Compose v2
2. Go 1.21+
3. Node.js 18+

## Setup Steps

1. Start development database:
```bash
docker-compose down -v
docker-compose up -d

sleep 5  # Wait for PostgreSQL to start
./scripts/init-test-db.sh
```

2. Build and start the server (API/control plane):
```bash
make build
WSIGN_AUTH_TOKEN_KEY=dev-secret-key \
  ./bin/wsignd --config configs/dev.yaml
```

3. Start the content server (data plane):
```bash
cd web
npm install
npm run dev
```

## Demo Flow

### 1. Display Registration (5 minutes)

1. Open web interface at http://localhost:3000/control
2. Browser interface loads and displays registration code (e.g., "BLUE-FISH")
3. In another terminal, use CLI to register display:
```bash
./bin/wsignctl display activate BLUE-FISH \
  --site headquarters \
  --zone lobby \
  --position main
```
4. Observe display transitioning to registered state

### 2. Content Management (5 minutes)

1. Add test content pointing to vite dev server:
```bash
./bin/wsignctl content add welcome \
  --url http://localhost:3000/demo/0001/welcome.html \
  --duration 10s \
  --config configs/wsignctl.yaml

./bin/wsignctl content add news \
  --url http://localhost:3000/demo/0001/news.html \
  --duration 15s \
  --config configs/wsignctl.yaml
```

2. Create content sequence:
```bash
./bin/wsignctl rule add \
  --display BLUE-FISH \
  --content welcome,news
```

3. Observe content rotation in browser interface

### 3. Health Monitoring (5 minutes)

1. View current display status:
```bash
./bin/wsignctl display list
```

2. Check content health metrics:
```bash
./bin/wsignctl content health welcome
```

3. Simulate error conditions:
   - Stop vite dev server
   - Observe automatic fallback
   - Watch error reporting in logs
   - Restart vite dev server and observe recovery

## Key Points to Demonstrate

1. Separation of Concerns
   - API server as control plane (8080)
   - Content served separately (3000)
   - Clear separation of duties

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
docker-compose down -v
```

## Additional Notes

- API server runs on port 8080
- Content served from port 3000 
- Health metrics persist in database
- Use --verbose flag with CLI for detailed output
