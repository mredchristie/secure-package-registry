FROM {{ index . "node_image" | default "node:23-alpine" }}

RUN apk add --no-cache git pnpm

WORKDIR /app

# Fetch specific commit to avoid mutable tag issues.
RUN git init && \
    git remote add origin {{ .repo_url }} && \
    git fetch --depth 1 origin {{ .commit_sha }} && \
    git checkout FETCH_HEAD

RUN pnpm install --frozen-lockfile

RUN pnpm build --filter ./{{ .build_dir }}

RUN mkdir -p /out && \
    cd {{ .build_dir }} && \
    pnpm pack --pack-destination /out

CMD ["sh", "-c", "ls /out/*.tgz"]
