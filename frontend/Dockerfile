FROM node:22-bookworm-slim@sha256:e9f565d32859f0cbfb52125eafe8be39390f24dac302dc3cdf566e8ba3ab21ee AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx/nginx-unprivileged:1.31.8-alpine@sha256:f90ef283a8a580912aa56315c3ed8f54cf50c2e89ed7e6fbf4c53368b55d712f
ENV API_HOST=api \
    API_PORT=8080 \
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1 \
    NGINX_ENVSUBST_FILTER="^(API_HOST|API_PORT|NGINX_LOCAL_RESOLVERS)$"
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
