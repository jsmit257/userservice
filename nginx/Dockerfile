FROM nginx:bookworm
COPY ./www/ /www
COPY ./userservice.conf /
ENTRYPOINT /bin/bash -c "envsubst </userservice.conf >/etc/nginx/nginx.conf && exec nginx -g 'daemon off;'"
