# Dev Kafka Topic Terraform

This configuration provisions the development Kafka topic for the realtime gateway.

## Prerequisites

- Terraform ≥ 1.5
- Kafka broker from `docker-compose.yaml` running locally (`localhost:9092`)

## Usage

```bash
cd deployments/terraform/dev
terraform init
terraform apply
```

Important variables already match the Compose stack:

- `kafka_bootstrap_servers` → `["localhost:9092"]`
- `topic_name` → `test-topic`
- `topic_partitions` → `3`
- `replication_factor` → `1`

Override any variable via `-var` or a `terraform.tfvars` file if needed.
