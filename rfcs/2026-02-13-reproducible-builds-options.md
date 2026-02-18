+++
title = "Reviewing Reproducible Builds Options"
author = ["readb5@cardiff.ac.uk"]
+++

## Summary

Considering the various options for implementing reproducible builds in our registry. The options considered include
auto-generating reproducible build scripts, manually creating reproducible build scripts (the Nix Way), and enforcing a
standardised project structure (the FDroid Way). After evaluating the pros and cons of each option, we recommend
following the Nix Way, as it provides a good balance between control over the build process and manageability of the
manual effort required.

## Problem

For diff checking and security purposes, we want to be able to check the integrity of packages in our registry. This
means that we need to be able to verify that the packages we are serving are the same as the packages that were built by
the original authors. To do this, we need to be able to reproduce the build process for each package, so that we can
compare the resulting build artifacts with the ones we are serving.

## Options Considered

### Initial Plan: The Auto-Generate Way

The initial plan for this project was to auto-generate reproducible build scripts for each package requested. This would
have been done by analyzing the package's build process and creating a script that could be used to reproduce the build.

#### Pros

- Would have been a one-stop solution for all packages, as the scripts would be generated on demand.

#### Cons

- Extremely complex, never implemented before.
- Many security considerations, as the generated scripts would need to be reviewed and tested to ensure they do not
  contain any breaking code.

### The Nix Way - Manual Reproducible Builds (By NixOS Contributors)

The Nix Way involves manually creating reproducible build scripts for each package, from NixOS contributors. This would
involve a more hands-on approach, where contributors would need to create and maintain the scripts for each package.

#### Pros

- More control over the build process, as contributors would be able to tailor the scripts to the specific needs of each
  package.

#### Cons

- Manual effort required per-package, scaling with each package added.

#### Note on Manual Effort Scaling

A good analogue for how difficult / time consuming this process is, is the NixOS project itself. NixOS has a staggering
120,000 packages (NixOS, 2026), and the NixOS contribute have been able to maintain reproducible builds for 91% of these
packages (Luj, 2024). NixOS has around 4000 contributors, and our organisation is much smaller than NixOS.

However, where NixOS maintains a large variety of packages, our registry only needs to maintain NPM, PyPI and Go
packages, which are all built using similar processes. This means that the manual effort required to maintain
reproducible builds for our registry is likely to be much lower than the effort required for NixOS. Additionally, the
number of packages is our registry is likely to be much smaller than the number of packages in NixOS, which further
reduces the manual effort required.

### The FDroid Way - Enforced Standardised Projects (By App Creators)

FDroid enforces a standardised project structure for Android apps, which allows for reproducible builds. App creators
are required to follow this structure, and FDroid provides tools to help with the build process.

#### Pros

- Standardised project structure makes it easier to create reproducible builds.
- App creators have more control over the build process, as they are responsible for following the standard

#### Cons

- Is dependant on app creators following the standardised project structure, which is unsuitable for our registry thats
  designed to be a mirror.

## Reccomendation

We will be following the Nix Way, since it is the only option that doesn't have critical flaws. The auto-generate way is
too complex and has too many security considerations, while the FDroid way is not suitable for our registry, since it
makes us dependant on package contributors. The Nix Way allows us to maintain control over the build process while still
ensuring that we can provide reproducible builds for our users. While it does require manual effort, It seems to be
quite manageable, especially considering the smaller scope of our registry compared to NixOS.

## References

NixOS, 2026. nixpkgs: The Nix Packages Collection. [online] GitHub. Available at:
[https://github.com/NixOS/nixpkgs](https://github.com/NixOS/nixpkgs]) [Accessed 14 February 2026].

Luj, 2024. Is NixOS truly reproducible? [online] Luj.fr. Available at:
[https://luj.fr/blog/is-nixos-truly-reproducible.html](https://luj.fr/blog/is-nixos-truly-reproducible.html) [Accessed
14 February 2026].
