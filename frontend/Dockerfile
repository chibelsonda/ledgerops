FROM node:22-bookworm-slim@sha256:e9f565d32859f0cbfb52125eafe8be39390f24dac302dc3cdf566e8ba3ab21ee AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx/nginx-unprivileged:1.30.6-alpine@sha256:ea4fd7ab5a8dc6348c18240bd3546f355f8373cd8f112580f7d8bd8ff5a74772
ENV API_HOST=api \
    API_PORT=8080 \
    NGINX_ENVSUBST_FILTER="^(API_HOST|API_PORT)$"
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
