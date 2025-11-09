FROM nginx:1.27-alpine

RUN addgroup -S web && adduser -S web -G web

COPY nginx.conf /etc/nginx/nginx.conf
COPY index.html /usr/share/nginx/html/index.html
COPY main.js /usr/share/nginx/html/main.js
COPY style.css /usr/share/nginx/html/style.css

RUN chown -R web:web /usr/share/nginx/html

USER web
