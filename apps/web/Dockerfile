FROM node:18-alpine AS builder

WORKDIR /app

COPY apps/web/package*.json ./
RUN npm install

COPY apps/web/ .
RUN npm run build

FROM node:18-alpine

RUN npm install -g serve

WORKDIR /app

COPY --from=builder /app/dist ./dist

EXPOSE 3000

CMD ["serve", "-s", "dist", "-l", "3000"]
