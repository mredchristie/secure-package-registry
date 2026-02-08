+++
title = "Reviewing OSS Registry Options"
author = ["readb5@cardiff.ac.uk"]
reviewers = ["cheongyx@cardiff.ac.uk"]
date = "2026-02-02"
+++

## Summary

Exploring the various open-source package registry options available, suitable for package ecosystems, primarily NPM,
PyPI and Go Modules, to determine the best fit for our needs.

The final decision made is using Gitea's package registry, which supports all three ecosystems while being more
lightweight and easier to manage than GitLab's package registry. S3 support is provided via MinIO.

## Problem

Since our system will rely heavily on these registries, making reliability, scalability, and robust automation APIs
critical requirements. Since most of our system is automatic, it is vital the registry's automation API's are well
documented, robust and of high quality.

Our architecture specifies S3-compatible storage backends for consistency across systems. While adaptable, we prefer to
maintain this standard.

The registries must be self-hosted and open-source to ensure full control and avoid vendor lock-in. Support for multiple
package ecosystems is advantageous to reduce operational overhead. Given that the vast majority of supply chain attacks
occur in NPM (87%) and PyPI (12%) (Ali, Ousat and Kharraz, 2025), support for these is essential. Support for the Go
ecosystem is also preferred, as it is this project's primary language, enabling us to verify the integrity of our own
dependencies and maintain system trust.

## Options Considered

### Verdaccio

Link: [https://github.com/verdaccio/verdaccio](https://github.com/verdaccio/verdaccio)

Supports S3?: Yes, via its Storage Plugin
[https://www.verdaccio.org/docs/plugin-storage/](https://www.verdaccio.org/docs/plugin-storage/)

Scalable?: Yes, Verdaccio focuses heavily on scalability, and has a number of features to support this, such as
clustering and load balancing.

Security?: Verdaccio has a number of security features, such as support for HTTPS, authentication, and access control.

Verdaccio is a lightweight private proxy registry for only the NPM ecosystem, it is widely adopted, has a large
community (17.5k stars on github), and is actively maintained. It requires zero config initally, which can be useful for
quickly setting up an MVP.

#### Biggest Advantages

- Most well maintained and widely adopted OSS registry for NPM.
- Fastest to MVP, requires zero config initially.

#### Biggest Disadvantages

- Only supports NPM ecosystem.

### GitLab Package Registry (OSS Tier)

Link: [https://docs.gitlab.com/administration/packages/](https://docs.gitlab.com/administration/packages/)

S3 Compatible?: Yes, via Object Storage integration
[https://docs.gitlab.com/administration/object_storage/](https://docs.gitlab.com/administration/object_storage/)

Scalable?: Yes, GitLab is designed to be scalable, and can handle large amounts of data and traffic. It is used by many
large organizations.

Security?: GitLab has a number of security features, such as support for HTTPS, authentication and access control.

GitLab's Package Registry is a feature of GitLab that allows users to host and manage packages for multiple ecosystems,
including NPM and PyPI. It is widely adopted, has a large community, and is actively maintained.

#### Biggest Advantages

- Since we are using GitLab for our git hosting, using its package registry would reduce the number of systems we need
  to maintain, and integrate well.
- Supports multiple ecosystems (NPM, PyPI).
- Wide adoption, large community, actively maintained.
- The best suited option for scalability, as GitLab is used by many extremely large organizations.

#### Biggest Disadvantages

- Heavyweight, complex to maintain/change.
- Not an isolated service, package registry is a small part of its service.
- Not exactly "open-source", some features are limited to paid tiers, and GitLab Inc. has been known to change licensing
  in the past. There may be legal implications to consider.
- No Go ecosystem support.

### Gitea Package Registry

Link: [https://docs.gitea.com/1.18/packages/packages/overview](https://docs.gitea.com/1.18/packages/packages/overview)

Supports S3?: Yes, it supports minio storage configuration, which can be used for S3 storage.
[https://docs.gitea.com/administration/config-cheat-sheet?\_highlight=minio#minio-storage-configuration-storage_minio](https://docs.gitea.com/administration/config-cheat-sheet?_highlight=minio#minio-storage-configuration-storage_minio)

Scalable?: Gitea is designed to be lightweight and easy to maintain, but may not be as scalable as other options like
GitLab.

Security?: Gitea has a number of security features, such as support for HTTPS, authentication, and access control.

Gitea is a lightweight self-hosted Git service, similar to GitLab but more lightweight. It has a package registry
feature that supports multiple ecosystems, including NPM, PyPI and Go (via its "Packages" feature). It is less widely
adopted than GitLab, but still has a decent community (19.3k stars on github), and is actively maintained.

#### Biggest Advantages

- More lightweight than GitLab, easier to maintain/change and deploy.
- Supports multiple ecosystems (NPM, PyPI).
- Supports Go ecosystem.

#### Biggest Disadvantages

- Less widely adopted than GitLab, smaller community.
- Documentation is smaller.
- Not an isolated service, package registry is a small part of its "all-in-one software development service", which may
  introduce bloat and increased attack surface.

### Pulp

Link: [https://pulpproject.org/](https://pulpproject.org/)

Supports S3?: Yes, can be configured to use S3 as a storage backend.
[https://docs.pulpproject.org/pulp_installer/objectstorage/](https://docs.pulpproject.org/pulp_installer/objectstorage/)

Scalable?: Yes, Pulp is designed to be scalable, and can handle large amounts of data and traffic.

Security?: Pulp has a number of security features, such as encryption, firewall, and access control.

The Pulp Project is an open-source platform for managing repositories of software packages and artifacts. It supports
multiple ecosystems, including NPM and PyPI. It is fully open source, and is supported by RedHat, arguably the world
most reputable open source vendor.

#### Biggest Advantages

- Fully open source, not limited because of paid tiers etc.
- Supports multiple ecosystems (NPM, PyPI)
- Documentation is high quality
- Supported by RedHat, a high quality open source vendor.

#### Biggest Disadvantages

- Doesn't support Go ecosystem.
- Not super widely adopted, smaller community, may be more fragile (Although RedHat is quite reputable).
- Written in Python which may be slower than other options. Although this may not be a big issue for our use case.

## Note on Access Control

All evaluated options support access control and authentication, our access control will be implemented at the network
level, so we don't need to worry about this aspect when comparing options. However, it is worth noting that tight access
control may actually be an impediment to implementing the proxy if we have to implement token cycling.

## Note on Package Upload

All of these options support package upload via the standardized APIs (NPM and PyPI), such as `npm publish`. This means
that if we need to switch to a different registry in the future, it should be relatively straightforward to do so
without having to change our automation code significantly. This is a significant advantage over proprietary solutions
such as JFrog Artifactory, which was excluded from consideration for this reason.

## Recommendation

**Gitea** is the recommended option.

It is the only solution besides GitLab that supports all three required ecosystems (NPM, PyPI, and Go) while being
significantly more lightweight and easier to manage. Although its community is smaller than GitLab's, it is active and
well-maintained with adequate documentation. S3 support is provided via MinIO.

All evaluated options are scalable and offer sufficient security. Crucially, their APIs for package upload/download
adhere to the standardized NPM and PyPI protocols, making future migration relatively straightforward (unlike
proprietary alternatives such as JFrog Artifactory, which was excluded for this reason).

While Gitea's package registry is part of a broader platform, initial deployment testing confirmed its resource
footprint is acceptable for our needs (only 180MB of memory used).

## Open Questions

1. Are any other ecosystems required apart from NPM, PyPI and Go?

2. Are there some advantages of using GitLab's package registry (considering we are already using GitLab for git
   hosting) that outweigh its disadvantages?

### Answered Questions

1. No, since we will have to manage our own GitLab instance & can't just reuse the University's. (via
   <cheongyx@cardiff.ac.uk>)

## References

Ali, A.S., Ousat, B. and Kharraz, A. (2025). Open Source, Open Threats? Investigating Security Challenges in Open-Source
Software. arXiv (Cornell University). doi:
[http://doi.org/10.48550/arxiv.2506.12995](https://doi.org/10.48550/arxiv.2506.12995)
