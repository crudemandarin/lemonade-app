# ---- build stage ----
FROM node:20-alpine AS build
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npx ng build --configuration production

# ---- runtime stage ----
FROM nginx:1.27-alpine
# The image runs envsubst on /etc/nginx/templates/*.template at startup.
# Defaults target Docker Compose; override both to point at another API.
ENV API_UPSTREAM=http://api:8080 \
    NGINX_RESOLVER=127.0.0.11
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/dist/lemonade-web/browser /usr/share/nginx/html

EXPOSE 80
