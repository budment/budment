---
title: Installation
description: Install the Budment CLI and SDK.
---

# Installation

Budment consists of two components:

- **CLI** — runs and manages test scenarios.
- **SDK** — defines test scenarios.

## CLI

### macOS / Linux

```bash
curl -fsSL https://budment.com/install.sh | bash
```

### Windows (PowerShell)

```bash
irm https://budment.com/install.ps1 | iex
```

### Docker

```bash
docker pull budment/budment:latest
```

### Verify

```bash
budment --version
```

## SDK

Install the SDK in your project:

```bash
npm install -D @budment/sdk
```

Import it in your scenario:

```typescript
import { http, sleep, random } from "@budment/sdk";
```
