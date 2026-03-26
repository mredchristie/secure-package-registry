FROM {{ index . "node_image" | default "node:23-alpine" }}

RUN apk add --no-cache git

WORKDIR /app

# Fetch specific commit to avoid mutable tag issues.
RUN git init && \
    git remote add origin {{ .repo_url }} && \
    git fetch --depth 1 origin {{ .commit_sha }} && \
    git checkout FETCH_HEAD

RUN npm ci

RUN npm run build

RUN mkdir -p /out && \
{{- if index . "githead_injection" }}
    node -e "const p=require('./package.json'); p.gitHead='{{ .commit_sha }}'; require('fs').writeFileSync('package.json', JSON.stringify(p, null, {{ index . "tab_size" | default "4" }}) + '\n')" && \
{{- end }}
    npm pack --pack-destination /out

CMD ["sh", "-c", "ls /out/*.tgz"]
