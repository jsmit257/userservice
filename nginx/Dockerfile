FROM nginx:bookworm
COPY ./www/ /www
# mount volumes over either of these directories to customize UI 
# look-and-feel or behavior
RUN /bin/sh -c "mkdir /www/js /www/css"
# defaults prevent 404s onload if there are no overrides
RUN /bin/sh -c "touch /www/js/custom.js /www/css/custom.css"
COPY ./userservice.conf /
ENTRYPOINT /bin/bash -c "envsubst </userservice.conf >/etc/nginx/nginx.conf && exec nginx -g 'daemon off;'"
