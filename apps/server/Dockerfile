# The Go server, with the card database it serves. Build from the repository
# root, with the card database checked out (git submodule update --init):
#
#   docker build -f apps/server/Dockerfile -t duel-masters-server .
#
# The server needs no C compiler (its SQLite driver is pure Go), so it is built
# as a static binary and runs on a distroless image: no shell, no package
# manager, a non-root user.

FROM golang:1.26 AS build

WORKDIR /src/apps/server

COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download

COPY apps/server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && mkdir /out/data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server
# The data folder belongs to the image's non-root user, so a new volume
# mounted there is writable.
COPY --from=build --chown=nonroot:nonroot /out/data /data

# The parts of the card database the server reads: the card data, the
# pictures and the set icons.
COPY resources/duelmasters_db/data/cards.json /resources/duelmasters_db/data/cards.json
COPY resources/duelmasters_db/data/images /resources/duelmasters_db/data/images
COPY resources/duelmasters_db/data/icons/sets /resources/duelmasters_db/data/icons/sets

ENV APP_ENV=production \
    HTTP_ADDR=0.0.0.0:8080 \
    DATA_DIR=/data \
    CARD_DATABASE_DIR=/resources/duelmasters_db

VOLUME /data
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=6s --start-period=10s --retries=3 \
    CMD ["/server", "healthcheck"]

ENTRYPOINT ["/server"]
