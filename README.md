# 🧠 Backend Tech Stack & Architecture - Monlai Project

This backend powers real-time FCM push notifications, async processing, and microservices for driver/customer engagement on Monlai. Built with scalability, observability, and modularity in mind.

---

## 🚀 Tech Stack

### 🏗 Core Frameworks & Language
- **Go (Golang)** – Backend language for all microservices
- **Gin** – Lightweight HTTP framework
- **Uber Fx** – Dependency injection framework for modular service composition

---

### 🧱 Microservices
Each of the following services runs independently in Docker containers:
- **`notify`** – Handles Firebase push notification logic
- **`driver`** – Driver management microservice
- **`customer`** – Customer-related microservice

---

### 💡 Architecture Patterns & Advanced Concepts

#### 🔄 Asynchronous Processing
- **AWS SQS**: Queue-based decoupling for driver and user notification triggers
- **Concurrent Workers**: Fixed goroutine worker pool listens on channel `notifyChan`

#### 🔁 Redis Patterns
- **Token Storage**: FCM tokens for each user/driver are stored using role-based Redis keys
- **(Optional)** Redis Strategy Pattern:
  - Used via `strategy.GetTokenStrategy(role)` abstraction
  - Enables plug-and-play support for customer/driver-specific token handling
- **Redis Pub/Sub / Locking** *(if implemented in extensions)*:
  - Considered for distributed coordination and event notification (pending code trace)

---

### 📦 Message Queuing
- **Amazon SQS**
  - `DRIVER_QUEUE_URL` for messages targeting drivers
  - `USER_QUEUE_URL` for messages targeting users
  - Ensures loose coupling and retry durability
  - Messages deleted **after successful enqueue or delivery**

---

### 📲 Push Notifications
- **Firebase Cloud Messaging (FCM)** – Delivered via the Go Admin SDK
- **Per-user token targeting** – Dynamically fetched from Redis
- **Dynamic message composition** – Structured payloads built per role/message context

---

### 📈 Observability
- **Prometheus Client (Go)** – Metrics exposed on `/notify/metrics`
- **Grafana Dashboard** – Connected via Prometheus scrape config to visualize:
  - Memory usage
  - Go runtime stats
  - Queue handler latency
- **API Gateway Proxying** – AWS Gateway routes public traffic to private `/metrics` for Prometheus

---

### 🔐 Secrets & Credential Management
- **Firebase Service Account JSON** – Injected securely into container via volume
- **`.env` file** – Centralized configuration for:
  - AWS credentials
  - Queue URLs
  - Redis URL
  - Firebase JSON path

---

### 🐳 Deployment
- **Docker**: Multi-container deployment (`driver`, `customer`, `notify`)
- **Docker Compose**: Local orchestration
- **GitHub Actions**:
  - CI/CD pipeline: builds, pushes to DockerHub, and SSH deploys to EC2
  - Writes service account and `.env` dynamically

---

### 🌐 Infrastructure
- **Amazon EC2** – Hosts all Docker containers
- **API Gateway (AWS)** – Publicly exposes FCM registration and metrics endpoints
- **Redis** – In-memory caching & token store (hosted or containerized)

---
