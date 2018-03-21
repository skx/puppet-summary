# First build puppet-summary
FROM alpine
RUN apk --no-cache add go git musl-dev
RUN go get -u github.com/skx/puppet-summary

# Now put it in a container without all the build tools
FROM alpine
WORKDIR /root/
COPY --from=0 /root/go/bin/puppet-summary .
ENV PORT=3001
EXPOSE 3001
VOLUME /app
ENTRYPOINT ["/root/puppet-summary", "serve", "-host","0.0.0.0", "-db-file", "/app/db1.sqlite", "-auto-prune" ]
