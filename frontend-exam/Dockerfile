FROM node:20-bookworm-slim AS deps

WORKDIR /app

COPY package*.json ./

RUN npm ci --legacy-peer-deps

FROM node:20-bookworm-slim AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules

COPY . .

ARG PUBLIC_API_URL=http://backend:8000
ENV PUBLIC_API_URL=${PUBLIC_API_URL}

RUN npm run build

FROM node:20-bookworm-slim AS runner

ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=4321

WORKDIR /app

COPY --from=builder /app/dist ./dist

USER node

EXPOSE 4321

CMD ["node", "./dist/server/entry.mjs"]
