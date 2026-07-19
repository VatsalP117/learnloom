FROM node:22-alpine AS web-builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY web ./web
ARG VITE_CLERK_PUBLISHABLE_KEY
ENV VITE_CLERK_PUBLISHABLE_KEY=$VITE_CLERK_PUBLISHABLE_KEY
RUN npm run build

FROM node:22-alpine

WORKDIR /app

COPY --chown=node:node package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force

COPY --chown=node:node bin ./bin
COPY --chown=node:node src ./src
COPY --chown=node:node --from=web-builder /app/web/dist ./web/dist
COPY --chown=node:node config.example.json README.md LICENSE ./

RUN mkdir -p /data && chown node:node /data

USER node

ENV NODE_ENV=production
ENV LEARNLOOM_HOME=/data

ENTRYPOINT ["node", "bin/learn.mjs"]
CMD ["run", "--config", "/app/config.json"]
