# Wrale Signage

Wrale Signage is an open-source digital signage management system designed for enterprise-scale deployments. It provides a simple yet powerful way to manage displays across multiple locations through a clean, domain-driven architecture.

## Overview

The system enables:
- Easy display setup using OAuth 2.0 Device Authorization Flow
- Flexible content delivery with robust caching
- Sophisticated content targeting based on location
- Resilient operation during network issues
- Simple administration through CLI and web interfaces

## Project Structure

The project follows a domain-driven design approach with clean architecture principles:

```
.
├── api/                  # API definitions and types
├── cmd/                  # Application entry points
│   ├── wsignd/          # Server binary
│   └── wsignctl/        # CLI tool
├── internal/            # Private implementation
│   ├── wsignd/         # Server implementation
│   └── wsignctl/       # CLI implementation
└── pkg/                # Public packages
```

## Development

### Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- Redis (optional, for caching)

### Building

```bash
go build ./cmd/wsignd     # Build server
go build ./cmd/wsignctl   # Build CLI tool
```

## License

Apache License 2.0 - See LICENSE file for details.