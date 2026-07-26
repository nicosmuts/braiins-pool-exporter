# Braiins API fixtures

These fixtures are synthetic, sanitized samples shaped from official Braiins
Pool documentation accessed on 2026-07-26 and a live read-only structural
validation checkpoint performed on 2026-07-26:

https://academy.braiins.com/braiins-pool/monitoring

No fixture is a raw account response. Replacements applied:

- account username replaced with `example-user`;
- worker names replaced with `worker-a`, `worker-b`, and `worker-offline`;
- reward and balance values replaced with synthetic BTC or satoshi values;
- payout destinations, transaction IDs, invoices, and preimages replaced with
  structurally obvious placeholders;
- timestamps replaced with public example timestamps that do not identify an
  operator.

Fixtures preserve documented and live-confirmed JSON field names, nesting,
nullable payout fields, worker state values, and numeric encoding choices.
No raw live response values remain.

Worker fixture names such as `worker-a` are synthetic labels. They are not raw
worker names and do not encode a site, person, device, or deployment mapping.
