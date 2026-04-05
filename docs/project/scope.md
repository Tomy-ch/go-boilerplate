# Project Scope

This document describes the **intended scope** of this project.

The goal is to clarify **what kind of systems this template is designed for**  
and **what use cases are not intended**.

## Target Team

This template assumes a team that understands the following technologies and practices:

- Architecture using Go + Echo + Fx + OpenAPI + sqlc
- Contract-driven API development using OpenAPI
- SQL-centric data access design
- Layered architecture (Onion / Clean Architecture)
- Development environments using Docker / Docker Compose
- Configuration management using `.env`
- Basic understanding of security boundaries

It is also assumed that the team includes members capable of making **initial architectural decisions**.  
As a guideline, this targets teams with **Tech Lead-level technical judgment**.

## Target Systems

This template is suitable for the following types of systems:

- Backend services for new products
- Applications in PoC to early scaling phases
- Systems requiring clear layer separation
- Projects requiring type-safe SQL management
- Systems with strong domain constraints (e.g., regulations, business rules)
- Backend systems intended for long-term maintenance
- Enterprise-grade domain-driven applications

This architecture assumes a **modular monolith based on Onion Architecture**.

## Non-Target Use Cases

This template may not be suitable for the following cases:

- Extremely small APIs implemented in a single file
- Rapid prototyping without architectural boundaries
- Ultra low-latency systems where abstraction must be minimized
- Systems designed as microservices from the beginning

## Architectural Assumptions

This project is based on a **modular monolith architecture**.

Characteristics:

- A single deployable application
- Clear module boundaries
- Layer separation based on Onion Architecture

It is possible to split into microservices in the future if needed,  
but that is **not the primary goal of this template**.
