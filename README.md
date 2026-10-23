# go-disposable-email

Refuse sign-ups from disposable (throwaway) email providers in Go.

- About 9,900 domains from
  [disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains)
  (CC0), embedded in the binary.
- Subdomains are caught (`a@x.mailinator.com`), lookalikes are not
  (`a@acmemailinator.com`).
- Mainstream providers (Gmail, Outlook, Yahoo, iCloud, Proton, …) are never
  refused, even if a wrong entry turns up upstream.
- No network at check time, no dependencies, about 75 ns per check.
- Works with any framework: it takes a string and returns an error.
- Optional background refresh, validated before use.

```
go get github.com/hymns/go-disposable-email
```

## Check an address

```go
import disposable "github.com/hymns/go-disposable-email"

if err := disposable.Check(email); err != nil {
	// err is disposable.ErrDisposable:
	// "disposable email addresses are not accepted; please use a permanent email"
}
```

`IsDisposable(email) bool` does the same without an error. Neither one
checks that the address is well formed; validate the format first.

In a `net/http` handler:

```go
func register(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if err := disposable.Check(email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// create the account
}
```

The same call works in any framework's handler; only the way you send the
400 changes. [`examples/`](examples/) has a runnable sign-up server for each:

| Framework | Example | Refuse with |
|---|---|---|
| net/http | [`examples/nethttp`](examples/nethttp/main.go) | `http.Error(w, err.Error(), http.StatusBadRequest)` |
| chi | [`examples/chi`](examples/chi/main.go) | `http.Error(w, err.Error(), http.StatusBadRequest)` |
| Gin | [`examples/gin`](examples/gin/main.go) | a `notdisposable` binding tag |
| Echo | [`examples/echo`](examples/echo/main.go) | `echo.NewHTTPError(http.StatusBadRequest, err.Error())` |
| Fiber v3 | [`examples/fiber`](examples/fiber/main.go) | `fiber.NewError(fiber.StatusBadRequest, err.Error())` |

```
cd examples/gin && go run .
curl -i -d '{"email":"a@mailinator.com"}' localhost:8080/register   # 400
```

Each example is its own module, so the library itself stays free of
dependencies.

It also fits a validator: with go-playground/validator, register
`disposable.IsDisposable` behind a custom tag.

```go
v.RegisterValidation("notdisposable", func(fl validator.FieldLevel) bool {
	return !disposable.IsDisposable(fl.Field().String())
})
// type SignUp struct { Email string `validate:"required,email,notdisposable"` }
```

## Your own rules

```go
checker := disposable.New(
	disposable.WithBlocked("new-throwaway.example"), // not listed upstream yet
	disposable.WithAllowed("partner.mailinator.com"), // listed, but you trust it
)
checker.Check(email)
```

The rules apply to subdomains too. At each level, from the full domain up,
a blocked domain refuses, an allowed one accepts, and a listed one refuses.
To give the package functions other rules, set
`disposable.Default = disposable.New(...)` once at start, before serving
requests.

## Keep the list current

The embedded list is as fresh as the release you built with. To follow
upstream daily, start a `Refresher`:

```go
r := &disposable.Refresher{
	Store:  disposable.FileStore("/var/lib/myapp/disposable.txt"), // optional
	Logger: slog.Default(),                                         // optional
}
go r.Run(ctx)
```

It fetches once at start and every 24 hours (`Interval`), with `If-None-Match`,
a 30-second timeout and a 10 MB cap. A download is used only if it has at
least 5,000 domains and every line is a host name; otherwise the list in use
stays. An accepted list is written to `Store`, so a restart does not need
GitHub.

For replicas, use a shared `Store`: each one reloads it every
`ReloadInterval` (1 hour), so a list one replica fetched reaches the others
within the hour. A `Store` is two methods:

```go
type pgStore struct{ db *sql.DB }

func (s pgStore) Get(ctx context.Context) (body, etag string, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT domains, COALESCE(etag, '') FROM email_blocklist WHERE id = 1`).Scan(&body, &etag)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	return body, etag, err
}

func (s pgStore) Put(ctx context.Context, body, etag string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO email_blocklist (id, domains, etag) VALUES (1, $1, NULLIF($2, ''))
		ON CONFLICT (id) DO UPDATE SET domains = EXCLUDED.domains, etag = EXCLUDED.etag`,
		body, etag)
	return err
}
```

## Updating the embedded list

```
go generate ./... && go test ./...
```

This downloads the upstream list, refuses it if it fails validation, and
rewrites `domains.txt` with the upstream commit in its header.

## Design

The decisions and the reasons for them are in [`docs/adr/`](docs/adr/), and
the map is in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Licence

The code is [MIT](LICENSE). The embedded list (`domains.txt`) is from
disposable-email-domains and is [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/).
