# Try the album API

Use PowerShell 7 and the Go version declared in `go.mod` or a compatible newer
release. From the repository root, start the server in one terminal:

```powershell
go run .
```

Keep it running and open another PowerShell terminal for the commands below.
The server listens on `localhost:8080` by default. If you set `LISTEN_ADDRESS`,
change `$base` to the server's HTTP address.

Albums are stored in memory. Every start loads sample albums with IDs `1`, `2`,
and `3`. Changes are lost when the process exits. This walkthrough uses the
separate ID `walkthrough-demo` and deletes it at the end. If that ID already
exists, choose another unused ID before starting; creation returns 409 for an
existing ID.

Run this block in order:

```powershell
$base = 'http://localhost:8080'
$id = 'walkthrough-demo'
$albumUrl = "$base/albums/$id"

# Check readiness. The response has status "ok".
Invoke-RestMethod "$base/health"

# Create an album. The response contains the album with price 29.99.
$album = @{
    id = $id
    title = 'Kind of Blue'
    artist = 'Miles Davis'
    price = 29.99
}
Invoke-RestMethod "$base/albums" -Method Post -ContentType 'application/json' -Body ($album | ConvertTo-Json)

# Read the newly created album by ID.
Invoke-RestMethod $albumUrl

# Replace it. PUT requires every field, including the unchanged ID.
$album.title = 'Kind of Blue (remastered)'
$album.price = 34.99
Invoke-RestMethod $albumUrl -Method Put -ContentType 'application/json' -Body ($album | ConvertTo-Json)

# Change only the price. Title and artist retain their previous values.
Invoke-RestMethod $albumUrl -Method Patch -ContentType 'application/json' -Body (@{ price = 24.99 } | ConvertTo-Json)

# Read a page and its total count. A fresh server now has four albums.
$page = Invoke-WebRequest "$base/albums?limit=2&offset=0"
$page.Content | ConvertFrom-Json
$page.Headers['X-Total-Count']

# Delete the album. A successful deletion returns 204 with no body.
$deleted = Invoke-WebRequest $albumUrl -Method Delete
$deleted.StatusCode

# Confirm it is gone. SkipHttpErrorCheck lets us inspect the expected 404.
$missing = Invoke-WebRequest $albumUrl -SkipHttpErrorCheck
$missing.StatusCode
$missing.Content | ConvertFrom-Json
```

The final response is 404 with `{"message":"album not found"}`. The sample
albums remain available. Stop the server with Ctrl+C when finished.

For field constraints and rejected requests, see the
[request contract](../README.md#getting-started). Open `/docs` on the running
server to explore the API interactively.
