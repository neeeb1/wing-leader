
# Wing Leader 🦆 

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://golang.org/) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/) [![Build and Test](https://github.com/neeeb1/rate_birds/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/neeeb1/rate_birds/actions/workflows/build-and-test.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/neeeb1/rate_birds)](https://goreportcard.com/report/github.com/neeeb1/rate_birds)

Finally, a community-powered ELO matchmaking system for birds.

Check it out at [wingleader.app](https://wingleader.app)

---

Wing Leader is a full-stack web application that allows users to vote on bird species in head-to-head matches and assigns ratings based on the [traditional ELO rating algorithim](https://en.wikipedia.org/wiki/Elo_rating_system). Deployed to Google Cloud Platform via a Github actions CI/CD platform.


Features
-   **Type-safe SQL queries utilizing [sqlc](https://github.com/sqlc-dev/sqlc?tab=readme-ov-file)**
-   **Database migrations via [goose](https://github.com/pressly/goose)**
-   **Structured logging using [zerolog](https://github.com/rs/zerolog) for observability**
-   **Lightweight static template generation using [HTMX](https://github.com/bigskysoftware/htmx)**
-   **Serverless compute via [Google Cloud Run](https://cloud.google.com/run?hl=en)**
-   **Session based vote tracking**
-   **IP-based rate limiting**
-   **Concurrent SQL safety**

## Motivation

This project was built as a learning excersie to familiarize myself with Golang backends, PostgreSQL, and CI/CD using Github Actions. More importantly, this project answers an age old question in an empiracal way using wisdom of the crowds - which is the best bird?

## 🏗 Quick Start - Local installation 

**Pre-requisites**
- [Docker](https://docs.docker.com/engine/install/)
- [Make](https://www.gnu.org/software/make/)
- [tailwindcss cli](https://tailwindcss.com/docs/installation/tailwind-cli)
- [Nuthatch api key](https://nuthatch.lastelm.software/)


1. Clone the repository
    ```bash
        git clone https://github.com/neeeb1/rate_birds
        cd rate_birds
    ```

2. Update env with your nuthatch api key and the DB url, user, and password.
    ```bash
    cp .env.example .env
    vim .env
    ```

2. Run the make command
    ```bash
    make compose-up
    ```

## 🌐 API Reference / Usage

### Core Endpoints

### Endpoints

#### `GET /api/loadbirds/`
Returns HTML fragment with two random bird cards and creates session.


#### `GET /api/scorematch/`
Records vote, updates ELO ratings, returns new bird pair.

**Query Params:**
- `winner`: `left` | `right`
- `leftBirdID`: UUID
- `rightBirdID`: UUID

**Headers:** `Cookie: sessionToken=<token>`

**Response:** New HTML fragment + new session cookie

**Error Codes:**
- `400`: Invalid parameters or missing session
- `401`: Session expired or already voted
- `429`: Rate limit exceeded

#### `GET /api/leaderboard/`
Returns top N birds by ELO rating as HTML table.

**Query Params:**
- `listLength`: Integer (1-1000)


### Health Checks

| Endpoint | Purpose | Use Case |
|----------|---------|----------|
| `GET /health/live` | Liveness probe | K8s/Cloud Run liveness |
| `GET /health/ready` | Readiness probe (DB ping) | K8s/Cloud Run readiness |

## 📈 Observability

### Metrics & Monitoring
- **Cloud Run Metrics:**
  - Request count and latency (p50, p95, p99)
  - Container instance count and CPU/memory utilization
  - Billable container time
  - Cold start frequency

- **Cloud SQL Metrics:**
  - Connection count and connection errors
  - Query execution time
  - Database size and storage utilization
  - Replication lag (if applicable)

- **Cloudflare Analytics:**
  - Global traffic distribution
  - Cache hit ratio
  - Threat detection and blocking
  - DNS query volume

### Logging
- **Format:** Structured JSON (zerolog)
- **Levels:** debug, info, warn, error
- **Fields:** timestamp, level, message, request_id, user_ip, bird_ids
- **Destination:** Cloud Logging (formerly Stackdriver)

## 🛠️ Tech Stack

### Backend
- **Language:** Go 1.25
- **Database:** PostgreSQL 17
- **SQL Codegen:** [sqlc](https://sqlc.dev/)
- **Migrations:** [Goose](https://github.com/pressly/goose)
- **Logging:** [zerolog](https://github.com/rs/zerolog)

### Frontend
- **Hypermedia:** [HTMX](https://htmx.org/)
- **Styling:** [Tailwind CSS v4](https://tailwindcss.com/)

### Infrastructure
- **Edge Network:** Cloudflare (DNS, DDoS, CDN)
- **Compute:** Google Cloud Run (serverless containers)
- **Database:** Cloud SQL (PostgreSQL 17)
- **Container Registry:** Google Artifact Registry
- **Monitoring:** Cloud Run metrics, Cloud SQL metrics, structured logging
- **CI/CD:** GitHub Actions

### External APIs
- **[Nuthatch API](https://nuthatch.lastelm.software/):** Bird species data

## 🎓 Learning Outcomes

This project demonstrates:

1. **Distributed Systems:** Handling concurrent database writes safely with transactions and row-level locking
2. **Cloud-Native Architecture:** Stateless design, auto-scaling, health checks, graceful shutdown
3. **Edge Computing:** Cloudflare integration for DDoS protection, CDN, and global distribution
4. **Layered Architecture:** Clean separation of concerns (API → Business Logic → Data)
5. **Testing Strategy:** Unit, integration, and concurrency tests with race detection
6. **CI/CD:** Automated testing and deployment pipelines with Docker and Artifact Registry
7. **Database Design:** Schema design, indexing, migrations, transaction management
8. **API Design:** RESTful principles, hypermedia (HTMX), session-based state
9. **Production Readiness:** Structured logging, metrics, error handling, security best practices
10. **Cost Optimization:** Serverless compute with pay-per-use pricing model


## 📝 Future Enhancements

- [ ] Static page generation for each bird with more info and conservation links
- [ ] CDN/image caching layer for bird images

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`go test ./... -race`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request


## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


## 🙏 Acknowledgments

- **Nuthatch API** for comprehensive bird species data
- **Boot.dev** for Go backend development curriculum
- **HTMX** community for modern hypermedia patterns
- **Go community** for excellent tooling ecosystem

---


<div align="center">

**Built with** ❤️ **by neeeb**


</div>
