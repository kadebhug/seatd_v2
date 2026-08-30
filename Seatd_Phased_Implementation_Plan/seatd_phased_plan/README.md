# Seatd Phased Implementation Plan

Start with [`00_MASTER_IMPLEMENTATION_ROADMAP.md`](./00_MASTER_IMPLEMENTATION_ROADMAP.md), then work through the numbered documents in order.

This plan is based on:
- the recommended Seatd target architecture (Go, PostgreSQL, Flutter, Next.js, transactional outbox, portable infrastructure);
- the current Seatd implementation/functionality reference;
- the requirement to avoid AWS-specific infrastructure unless there is a concrete need.

The order is intentional: the operational truth model and command semantics are dependencies for offline sync, realtime, clients, analytics, and integrations.
