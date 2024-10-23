# CCPv2

[![GitHub Release](https://img.shields.io/github/v/release/artefactual-labs/ccp?style=flat-square)](https://github.com/artefactual-labs/ccp/releases/latest)

## Introduction

CCPv2 is an exploratory initiative being developed in parallel with the ongoing
maintenance of Archivematica. It aims to provide a streamlined AIP (Archival
Information Package) creation tool, staying aligned with Archivematica's
standards while radically simplifying the architecture.

CCPv2 is not a framework for building a future project but a space for
experimentation and learning. The outcome is uncertain — it may be rejected,
used for inspiration, or integrated with existing tools. The focus is on
exploring new approaches to modularity, flexibility, and scalability in digital
preservation, addressing common challenges faced in enterprise deployments.
Developed iteratively in small, manageable batches, CCPv2 allows for continuous
refinement and minimizes risk as new features are introduced.

## Installation

Currently, we only support Kubernetes deployments using the [prod overlay] from
the [deploy branch]. To set up CCPv2 for testing purposes, use the following
command:

```
kubectl apply -k overlays/prod
```

> [!NOTE]
> We are working to make deployments more accessible. Future installation
> methods being explored include a Helm chart for more scalable deployments and
> a self-contained binary file suitable for simpler, lightweight scenarios.

[prod overlay]: https://github.com/artefactual-labs/ccp/tree/deploy/overlays/prod
[deploy branch]: https://github.com/artefactual-labs/ccp/tree/deploy

## Features

### Core compatibility with Archivematica

CCPv2 creates standard Archivematica AIPs and aims to remain fully compatible
with Archivematica, producing equivalent packages. It is capable of ingesting
the same metadata found in Archivematica's standard transfers, ensuring seamless
integration within existing workflows.

### Modern web interface

The user interface is being redesigned using modern web standards to improve
scalability and provide reactivity during processing, allowing for smoother
management of large volumes of packages and real-time updates during operations.

### API-driven

As a core design principle, all functionality in CCPv2 is available via its API
([docs]). Surrounding tools like the web interface and command-line interfaces
(CLIs) are built on top of this API, ensuring consistency and flexibility across
different interaction methods.

[docs]: https://buf.build/artefactual/archivematica/docs/main:archivematica.ccp.admin.v1beta1

### Pluggable storage interfaces

CCPv2 aims to be storage-agnostic, enabling integration with both custom
solutions and widely used systems like the Archivematica Storage Service, while
also providing support for additional storage interfaces in the future.

### Faster release cycle

We prioritize frequent releases to quickly iterate on ideas and gather feedback.
Rather than waiting for every feature to be perfect, we prefer to release often,
even with partially completed features. We employ continuous delivery via
ArgoCD, which automates the deployment of updates to testing environments as
soon as changes are made, allowing for rapid testing and validation of new
ideas.

### Efficient deployments

CCPv2 is designed for lightweight deployments with minimal dependencies. Unlike
Archivematica, it does not require Nginx, Elasticsearch, Gearman, or Gunicorn,
simplifying deployment across various environments.

## CCPv2 vs Alternatives

### Archivematica

CCPv2 currently tracks the Archivematica codebase closely, frequently merging
updates from Archivematica's main branch to stay aligned with its core
workflows. Both CCPv2 and Archivematica use the same workflow document and
client modules, ensuring compatibility for AIP creation. However, CCPv2 diverges
in several key areas: it removes built-in search capabilities and replaces
Archivematica's  storage and transfer location integrations with a pluggable
interface, providing more flexibility in how storage is managed. Additionally,
CCPv2 introduces a modern API and web interface to simplify the user experience
and streamline development.

### a3m

a3m took a more radical approach compared to CCPv2, diverging from Archivematica
in several significant ways. While both focus on AIP creation, a3m removed
additional functionalities like reingest and DIP (Dissemination Information
Package) creation, which CCPv2 has not yet explored. a3m also lacked a web
interface, whereas CCPv2 introduces a modern interface to simplify user
interaction. a3m remained in the Python ecosystem, while CCPv2 adopts Go for a
simpler, more scalable solution, using Go's opinionated design and concurrency
to meet its goals.

### Enduro

CCPv2 and Enduro don't necessarily overlap, as they focus on different stages of
the digital preservation process. Enduro is primarily focused on validation and
building infrastructure to support custom preprocessing workflows at scale,
allowing different institutions to tailor their workflows before preservation.
In contrast, CCPv2 is focused on preservation tasks, specifically creating and
managing AIPs to ensure digital assets are properly handled and transferred to
long-term storage systems. Enduro already integrates with Archivematica and a3m,
and it could easily extend support to CCPv2.

### CCPv1

CCPv1 was a strategic proposal for iteratively rewriting Archivematica's
MCPServer to modernize its architecture while keeping Archivematica in its v1
form. Though this approach was not adopted due to differing short- and
medium-term priorities, it laid the foundation for CCPv2. Now, CCPv2 explores
new ways to simplify and enhance Archivematica, focusing on introducing a new
API, web interface, and pluggable storage system, while staying aligned with the
core standards of Archivematica. For those interested in exploring CCPv1
further, a [v1] tag is available.

[v1]: https://github.com/artefactual-labs/ccp/releases/tag/v1

## Contributing

We welcome contributions to CCPv2! Please see the [contributing guide] for
information on how to get involved. Whether it's reporting bugs, contributing
code, or improving documentation, we value your input!

[contributing guide]: https://github.com/artefactual-labs/ccp/blob/qa/2.x/CONTRIBUTING.md

## License

CCPv2 is licensed under the AGPLv3. See the [LICENSE] file for more details.

[LICENSE]: https://github.com/artefactual-labs/ccp/blob/qa/2.x/LICENSE
