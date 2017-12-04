FROM alpine
RUN apk add --update --no-cache bash
RUN apk add --no-cache curl
ADD watch.sh /
ENV HOME=/
CMD ["/watch.sh"]
