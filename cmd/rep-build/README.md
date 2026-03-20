> [!WARNING]
> DO NOT push code/make PR's on GitHub, this is a mirror of the original codebase (On GitLab), which is
> private. Please clone the original repository and push your code there.

# Reproducible Build Component

This section of the codebase is responsible for managing the reproducible build process for the packages in our
registry.

Since we are using GitHub Actions for this part of our CI/CD pipelines. This code is mirrored on GitHub
[https://github.com/BenjyRead/Registry-Reproducible-Builds](https://github.com/BenjyRead/Registry-Reproducible-Builds),
this repository is used for generating build artifacts and such aswell.

## How to use generate_directory.go

This utility script is used to generate the directory structure for a package based on its name and version. It can be
run with either a single dependency string or separate name and version flags.

Usage: `go run generate_directory.go -dep "@sveltejs/adapter-auto:^7.0.0"` Or:
`go run generate_directory.go -name "@sveltejs/adapter-auto" -version "^7.0.0"`
