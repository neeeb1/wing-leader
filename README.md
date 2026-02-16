
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


# 🏗 Local installation 

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

## 🌐 API Reference

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
- **Framework:** Standard library `net/http`
- **Database:** PostgreSQL 17
- **SQL Codegen:** [sqlc](https://sqlc.dev/)
- **Migrations:** [Goose](https://github.com/pressly/goose)
- **Logging:** [zerolog](https://github.com/rs/zerolog)

### Frontend
- **Hypermedia:** [HTMX](https://htmx.org/)
- **Styling:** [Tailwind CSS v4](https://tailwindcss.com/)
- **No JavaScript framework** (HTMX handles interactivity)

### Infrastructure
- **Edge Network:** Cloudflare (DNS, DDoS, CDN)
- **Compute:** Google Cloud Run (serverless containers)
- **Database:** Cloud SQL (PostgreSQL 17)
- **Container Registry:** Google Artifact Registry
- **Monitoring:** Cloud Run metrics, Cloud SQL metrics, structured logging
- **CI/CD:** GitHub Actions

### External APIs
- **[Nuthatch API](https://nuthatch.lastelm.software/):** Bird species data

## Technologies Used
| Technology         | Description                                                               | Link                                                    |
| :----------------- | :------------------------------------------------------------------------ | :------------------------------------------------------ |
| **Go**             | Core language for the API backend, ensuring performance and concurrency.  | [golang.org](https://golang.org/)                       |
| **PostgreSQL**     | Robust relational database for storing bird and rating data.              | [postgresql.org](https://www.postgresql.org/)           |
| **`sqlc`**         | Generates type-safe Go code from SQL queries, improving development.      | [sqlc.dev](https://sqlc.dev/)                           |
| **HTMX**           | Enables dynamic, interactive HTML interfaces with minimal JavaScript.     | [htmx.org](https://htmx.org/)                           |
| **`go-dotenv`**    | Simple environment variable loading from `.env` files.                    | [github.com/joho/godotenv](https://github.com/joho/godotenv) |
| **Nuthatch API**   | External REST API providing comprehensive bird species data.              | [nuthatch.lastelm.software](https://nuthatch.lastelm.software/) |

## License
This project is open-source and licensed under the MIT License.


---

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![HTMX](https://img.shields.io/badge/HTMX-3069C7?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB2aWV3Qm94PSIwIDAgMjQgMjQiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHBhdGggZmlsbD0iI2ZmZiIgZD0iTTEwLjI2OSAyLjAyNWwtLjAzLS4wNzUtLjAwNC0uMDA4LS41NzYtMS42NDktLjAwNC0uMDA4LS4wMy0uMDc0LS4xNS0uMzcySDB2MjRsMTkuMDIzLjA1NS4wMDMtLjAwNS4wNzgtLjIwNy4wMjctLjA3NC4wMDQtLjAwOEw4Ljg4NCAyNC4wMDlsLS4wMDUtLjAwOHMtMy42LS4zMzUtNC43NzItLjQ0N2ExLjE5LjExOSAwIDAgMS0uNjEyLS43MDZjLS4xNjQtLjU2NC4xNDQtMS4yMDggLjcwOC0xLjM3Mi41NjQtLjE2NC43MDgtLjI1Mi44Mi0uNjc2LjE0NC0uNjE2LTIuMDA0LS44NzItMi43NTItLjk2LTEuNTU2LS4xODgtMS43MTItLjI5Ny0xLjk2OC0uMzkyLS44Mi4wMzYtMS42NC4wMzYtMi41Ni4zNzYtLjUwOC4yMDQtMS4wMy40ODctMS41NzYuODk5LS4xMy4xMDgtLjI1Mi4yMjgtLjM4OC4zNjQtLjIwOC4xODgtLjQyNC4zNzItLjY3Ni41NDQtLjYxMi40NDQtMS40OTYuNjI4LTIuMjQ0LjYwNC0uOTIyLS4wMzYtMS43ODQtLjI5Ny0yLjM5Mi0uNzU3LS41NDgtLjQzMi0uNzg0LTEuMDgtLjYyLTEuNTA5LjE2NC0uNDE2LjQ2OC0uNjI4LjcyOC0uNjg4LjI0LS4wNjQuNjUtLjA2NC45MDguMDcyLjE5Mi4xMDIuNTUyLjIyOC44NTYuNDA0LjI5Mi4xOC42MDQuMzY4LjkyOC41NC42NDQuMzQ4IDEuNjY4LjUzMiAyLjg1Mi4zMTYuNzk2LS4xNDggMS42ODQtLjQ3NiAyLjM4LS42OC41NzItLjE2LjkyLjIzNi45OC42NjQuMDY4LjQzMi0uMDcyLjcyLS40NTYuODQtLjM4NC4xMjgtLjg3Mi4wNTItMS4xMDgtLjA2OC0uOTg4LS41MDQtMi4xMDQtLjgxNi0zLjIxMi0uOTA4LS4wNzYtLjAwNC0uMDc2LS4wMDQtLjA5Mi0uMDA0aC02LjMzMnYtNS45Nmg1LjE0NmMzLjQ0LS4zNTIgNi4zMjgtLjQ3NiA4LjcwNC0uNzc2IDYuMjQtLjYxNiA4Ljg0OC0xLjgyIDguODQ4LTUuNjYgMC0zLjgxNi0yLjg4LTUuMjcyLTguNTQ4LTUuNzY0LTMuMjktLjI4NC03LjQ1Mi0uMzY4LTEwLjk1Ni0uMzg0LS4wMTYtLjAwNC0uMjctLjAwNC0uNjEyLS4wMDVsLS4wNzYtLjAwNGwtLjk3Ni0uMDQ4YTEuMTkyLjQ3NiAwIDAgMS0uNzg4LS40NzYgMS4wNiAxLjA2IDAgMCAxIC4wNDgtLjgyMWMuMDk2LS4zMTIuMjk2LS40NzYuNTg4LS41NDRsLjM5Ni0uMDk2LjcwNC0uMDQ0Yy44Mi0uMDk2IDEuOTM2LS4xNTEgMy4yMTYtLjE1MXoiLz48L3N2Zz4=)](https://htmx.org/)
[![Readme was generated by Dokugen](https://img.shields.io/badge/Readme%20was%20generated%20by-Dokugen-brightgreen)](https://www.npmjs.com/package/dokugen)
