/*
Package workhub implements the WorkerService API.

The Hub manages interactions with distributed workers, including:
  - Enqueuing batches of tasks submitted by producers.
  - Allowing workers to grab available batches via long-polling.
  - Receiving results from workers upon batch completion.
  - Providing a mechanism for producers to wait for and receive batch results.
  - Tracking worker state (Idle, Working).
*/
package workhub
