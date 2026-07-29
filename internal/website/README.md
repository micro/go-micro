# Go Micro website

The Go Micro documentation site is built with [Hugo](https://gohugo.io/) and the
[Docsy](https://www.docsy.dev/) theme. Site content lives in `content/en`, while
project-specific templates and styling live in `layouts` and `assets`.

## Local development

Hugo Extended, Go, and Node.js are required. From this directory, run:

```sh
npm ci
npm run serve
```

The production build used by GitHub Pages is:

```sh
npm ci
hugo --gc --minify
```

Docsy is pinned as a Hugo module in `go.mod`; no theme submodule checkout is
required. Deployment is handled by `.github/workflows/website.yml` from the
repository root.
