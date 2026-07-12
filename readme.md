# Blaster

> **The Minimum Safety Net That Backend Should Have.**

Blaster is an OpenAPI-driven API testing engine that automatically generates, executes, and validates backend tests with almost zero manual setup.

Instead of manually writing hundreds of API test cases, Blaster understands your API, builds an execution plan, prepares dependent data, performs stateful testing, fuzzes requests, validates responses, and produces CI-ready reports automatically.

---

## Why Blaster?

Modern backend teams usually have one of these problems:

- APIs are released without automated tests.
- Writing API tests is repetitive and expensive.
- Load testing tools don't understand API relationships.
- Fuzzers generate invalid requests.
- CI pipelines become slower as projects grow.

Blaster solves these problems by understanding your OpenAPI specification and turning it into executable tests automatically.

---

## Quick Start

Initialize a project.

```bash
blaster init
```

Parse your OpenAPI specification.

```bash
blaster parse
```

Run the complete testing pipeline.

```bash
blaster run
```
---
