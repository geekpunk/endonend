FROM node:22-alpine AS build
WORKDIR /src
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
# VITE_API_URL is baked in at build time, per Vite convention: the browser
# talks directly to the backend's published port, not through this
# container, so this must be reachable from wherever the browser runs, not
# just from inside the compose network.
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
