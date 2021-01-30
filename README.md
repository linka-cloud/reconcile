# Reconcile

Reconcile is a small module implementing the reconcile pattern.
It provided the three main elements:
- 2 Storage interfaces providing persistence and observability: Read/Watch and Write
- A Cache, an in memory read-only storage holding the already fetched resources
- A Reconciler pattern implementation, watching the cache and called on every resource event
