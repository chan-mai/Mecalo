FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mecalo ./cmd/mecalo \
	&& mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mecalo /mecalo
COPY --from=build --chown=65532:65532 /out/data /data
ENV MECALO_LISTEN=:8080 \
	MECALO_DATABASE_URL=/data/mecalo.db
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/mecalo"]
CMD ["serve"]
