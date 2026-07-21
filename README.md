```
                     _______   ________  __         ______   __      __
                    /       \ /        |/  |       /      \ /  \    /  |
                    $$$$$$$  |$$$$$$$$/ $$ |      /$$$$$$  |$$  \  /$$/
                    $$ |__$$ |$$ |__    $$ |      $$ |__$$ | $$  \/$$/
                    $$    $$< $$    |   $$ |      $$    $$ |  $$  $$/
                    $$$$$$$  |$$$$$/    $$ |      $$$$$$$$ |   $$$$/
                    $$ |  $$ |$$ |_____ $$ |_____ $$ |  $$ |    $$ |
                    $$ |  $$ |$$       |$$       |$$ |  $$ |    $$ |
                    $$/   $$/ $$$$$$$$/ $$$$$$$$/ $$/   $$/     $$/


```
![Go](https://img.shields.io/badge/Go-1.22-blue?logo=go)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
![Status](https://img.shields.io/badge/status-in_progress-brightgreen)
![Project](https://img.shields.io/badge/project-personal-blueviolet)

## Table Of Content

- [Description](#description)
- [Highlights](#highlights)
- [Installation & Usage](#installation--usage)
- [Project Structure](#project-structure)
- [Limitations](#limitations)
- [Milestones](#milestones)

## Description

Relay is a Webhook debugging tool. It gives developers a public URL that forwards to their local server, captures every request/response exchange, and lets them replay any captured request without re-triggering the original event.

This matters when integrating third party webhook providers like GitHub, Paystack, Stripe into your application. For such integration, during development, you'd need a public URL to give them so they can send requests and when a handler crashes on a real payload, you don't want to trigger a new payment to reproduce the error. Relay allows you to replay the same event as many times as you want.

## Highlights

- **Custom multiplexing protocol** - single tunnel connection carries many concurrent requests using a length-prefixed frame format inspired by HTTP/2
- **Byte-level HTTP parsing** - built directly on `net.Conn`, no `net/http` in the tunnel or capture layers
- **Debugging journal** - [lessons-learned.md](./lessons-learned.md) documents bugs I hit, wrong assumptions that produced them, and what I learned
- **Milestone reflections** - [milestones/](./milestones/) analyses design tradeoffs after each major piece

## Installation & Usage

You will need Go 1.22 or later. You can check with:

```bash
go version
```

Clone the repo:

```bash
git clone https://github.com/Ajibose/Relay.git
cd Relay
```

That's it - no external dependencies to fetch. `go mod tidy` if you want to be sure:

```bash
go mod tidy
```

### Running locally

Relay has two binaries: one, `relayd` (the public server) and the other being `relayc` (the client that dials in from your machine). You'll also need any local HTTP server on `:8080` - that's what `relayc` will forward requests to.

**Ports 5000, 5001 and 8080 should be free on your machine**

Open four terminals.

**Terminal 1** - your local HTTP server. Anything on port 8080. For a quick test:
Run any local server on port :8080
e.g

```bash
python3 -m http.server 8080
```

or write your own

**Terminal 2** - start `relayd`:

```bash
go run ./cmd/relayd
```

`relayd` listens on `:5000` (for `relayc` to dial in) and `:5001` (for visitor requests).

**Terminal 3** - start `relayc`:
```bash
go run ./cmd/relayc
```
`relayc` dials `relayd` on `:5000` and holds the tunnel open.

**Terminal 4** - send a request:

```bash
curl http://localhost:5001/hello
```

You should get the response from your local server on `:8080`.

To test multiplexing (many concurrent requests over one tunnel):

```bash
for i in 1 2 3 4 5; do curl -s http://localhost:5001/ & done; wait
```

## Project Structure

```
.
├── cmd
│   ├── relayc
│   │   └── main.go
│   └── relayd
│       └── main.go
├── debugging_journal.md
├── double-close-race.png
├── go.mod
├── internal
│   └── tunnel
│       ├── frame.go
│       ├── frame_test.go
│       └── mux.go
├── milestones
│   └── m2-reflection_notes.md
└── README.md
```

## Limitations

Being built in stages. Known gaps in the current implementation:

- **Single tunnel only** - `relayd` accepts one `relayc` connection at a time for now. Multi-tenant subdomain routing is planned for M7(Milestone 7).
- **No flow control** - a slow visitor can cause response bytes to queue up in memory. Long-running scenarios with many slow users would degrade throughput or exhaust memory. HTTP/2's WINDOW_UPDATE mechanism is planned as the fix, parked for a future revisit.
- **Best-effort cleanup on tunnel failure** - when the tunnel breaks, active streams rely on process exit for FD cleanup. Would need coordinated shutdown (e.g. via `context.Context`) if the relay were to auto-reconnect.
- **No TLS or authentication yet** - production deployment (planned for M8) will require both.

## Milestones

- [x] M0 - Two-party byte relay
- [x] M1 - Single-request HTTP tunnel
- [x] M2 - Multiplexed tunnel with custom frame protocol
- [ ] M3 - Request capture (SQLite)
- [ ] M4 - Inspector API + minimal UI
- [ ] M5 - Replay
- [ ] M6 - Real-time inspector (SSE)
- [ ] M7 - Subdomain routing (multi-tenant)
- [ ] M8 - TLS + public deployment
