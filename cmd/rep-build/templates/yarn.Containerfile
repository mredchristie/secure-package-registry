FROM {{ index . "node_image" | default "node:23-alpine" }}

RUN apk add --no-cache git

WORKDIR /app

# Fetch specific commit to avoid mutable tag issues.
RUN git init && \
    git remote add origin {{ .repo_url }} && \
    git fetch --depth 1 origin {{ .commit_sha }} && \
    git checkout FETCH_HEAD

RUN yarn install --frozen-lockfile

RUN yarn build

RUN mkdir -p /out && \
    yarn pack --filename /out/package.tgz

CMD ["sh", "-c", "ls /out/*.tgz"]
