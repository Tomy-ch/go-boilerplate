# Project Scope

This document describes the intended scope of this boilerplate.

The goal is to clarify **what kinds of systems this template is designed for** and **what it is not intended to support**.

## Assumed Team

This template assumes a team that understands the following technologies and practices:

- Go + Echo + Fx + OpenAPI + sqlc architecture
- Contract-driven API development using OpenAPI
- SQL-centered data access design
- Layered architectures (Onion / Clean Architecture)
- Docker / Docker Compose development environments
- .env configuration management
- Basic security boundary awareness

It is assumed that the team can make **initial architectural decisions**, typically at a **Tech Lead level**.

## Intended System Types

This template is designed for the following types of systems:

- Backend services for new products
- PoC to early scaling phase applications
- Systems requiring strict layer separation
- Projects requiring type-safe SQL management
- Systems with strong domain constraints (legal rules, business rules, etc.)
- Backend systems expected to be maintained long term
- Enterprise-style domain-driven applications

The architecture assumes a **modular monolith based on Onion Architecture**.

## Not Intended For

This template may not be suitable for the following cases:

- Extremely small APIs implemented in a single file
- Rapid prototyping without architectural boundaries
- Ultra-low latency systems requiring minimal abstraction
- Systems designed primarily as microservices

## Architecture Assumption

This boilerplate assumes a **modular monolith architecture**.

Characteristics:

- Single deployable application
- Clear module boundaries
- Layer separation through Onion Architecture

Microservices can be introduced later if necessary, but they are **not the primary goal of this template**.
