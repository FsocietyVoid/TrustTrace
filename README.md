# TrustTrace 🛡️

**Blockchain-Backed SRE Observability & Zero-Trust SLA Enforcement**

TrustTrace is a distributed, zero-trust observability platform designed to eliminate B2B dispute friction. By decoupling bulk telemetry storage from cryptographic state anchoring, TrustTrace provides irrefutable, ledger-backed proof of uptime, latency, and SLA compliance.

Standard monitoring tools (like Datadog or New Relic) rely on mutable databases controlled by the service provider. TrustTrace solves this by cryptographically signing edge node telemetry and anchoring 10-minute Merkle root proofs directly to a blockchain. 

## 🏗️ System Architecture

TrustTrace utilizes a clean, modular architecture separated into three primary execution planes:

1. **Edge Probers (`cmd/prober`)**: 
   Lightweight, distributed Go binaries deployed across multiple global regions. They utilize a non-blocking worker pool to ping target endpoints and cryptographically sign the results using Ed25519 node keys.
2. **Consensus & Ingestion Engine (`cmd/consensus`)**: 
   A centralized control plane that aggregates signed metrics via gRPC. It applies multi-region quorum logic (e.g., 2-of-3 nodes must agree) to eliminate localized routing errors and false positives.
3. **Cryptographic Notary (`cmd/notary`)**: 
   The background engine that chunks granular telemetry stored in a time-series database (ClickHouse/TimescaleDB) into 10-minute intervals. It builds a Merkle Tree of the data and commits the 32-byte root hash to the blockchain ledger for financial-grade settlement.

## 🚀 Quick Start

### Prerequisites
* Go 1.22+
* Docker & Docker Compose
* Make

### Installation

1. **Clone the repository:**
   ```bash
   git clone (https://github.com/FsocietyVoid/trusttrace.git)
   cd trusttrace