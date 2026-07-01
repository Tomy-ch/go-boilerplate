# debugmode

English | [日本語](README.ja.md)

Enables Echo debug mode in development environments only.

## Role

Verbose debug output is useful while developing but leaks internal detail in production. Gating it behind an environment check in one place keeps that decision explicit and centralized, so the rest of the stack never has to branch on environment to decide how much to expose.
