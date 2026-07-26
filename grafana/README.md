# Grafana dashboard

The reusable dashboard is planned for Milestone 06.

- UID: `braiins-pool-exporter`
- Title: `Braiins Pool Exporter`

It will use only verified metrics emitted by this exporter and will cover
account hashrates, worker state and trends, last-share age, rewards, payouts,
API health, and freshness.

No dashboard JSON is included in Milestone 00 because API-derived metrics do
not exist yet. Speculative queries would create a misleading compatibility
contract.

The public dashboard will not depend on Avalon telemetry, Smuts Tech worker
names or IP addresses, Bitcoin price providers, wallet monitoring, Kubernetes
metrics, or workshop-specific Grafana provisioning.
