# Wrale Signage [ALPHA]

Wrale Signage aims to be an open-source digital signage management system with a focus on simplicity and reliability. Currently in early development.

## Current Status

Basic functionality is implemented:
- Display registration and management via CLI
- Content event tracking in PostgreSQL
- Basic health monitoring
- Web interface for display simulation

## Project Structure

```
.
├── api/                  # API types
├── cmd/                  # Binaries
│   ├── wsignd/          # Server
│   └── wsignctl/        # CLI
├── internal/            # Implementation
└── web/                # Display interface
```

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Node.js 18+ (for web interface)
- Docker (for development database)

### Quick Start

1. Start development database:
```bash
make test-deps
```

2. Build binaries:
```bash
make build  # Creates bin/wsignd and bin/wsignctl
```

3. Run tests:
```bash
make test
```

See `/docs/demos/0001_basic_setup_and_content.md` for complete setup guide.

## Contributing

Project is in early stages. APIs and interfaces may change significantly.

## License

Apache License 2.0