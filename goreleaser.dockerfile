FROM scratch

LABEL maintainer="Karim Radhouani <medkarimrdi@gmail.com>""
LABEL documentation="https://orbrs.kmrd.dev"
LABEL repo="https://github.com/karimra/ouroboros"

COPY orbrs /app/orbrs
ENTRYPOINT [ "/app/orbrs" ]
CMD [ "--help" ]
