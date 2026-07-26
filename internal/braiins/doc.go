// Package braiins contains the minimal transport and wire-schema boundary for
// the official Braiins Pool API.
//
// Milestone 01 intentionally stops at request construction, status handling,
// JSON decoding, and verified wire types. Polling, caching, retry/backoff, and
// Prometheus collectors belong to later milestones.
package braiins
