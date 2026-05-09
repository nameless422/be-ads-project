# Quickstart

## Frontend

```bash
cd fe
npm install
npm run dev
```

## Production Serving Through Backend

```bash
cd be
make up
```

Then open:

```text
http://127.0.0.1:8080/bi
```

## Before Submitting UI Changes

```bash
cd fe
npm run build

cd ../be
go test ./...
```

