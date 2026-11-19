# Revalidate the OpenAPI document

Use PowerShell 7. Start the service with `go run .` from the repository root,
then run this block in another terminal. If you set `LISTEN_ADDRESS`, change
`$base` to the server's HTTP address.

```powershell
$base = 'http://localhost:8080'
$documentUrl = "$base/swagger/doc.json"

# Fetch the document and keep its quoted ETag exactly as returned.
$document = Invoke-WebRequest $documentUrl
$etag = $document.Headers['ETag'][0]
[pscustomobject]@{
    StatusCode = $document.StatusCode
    ETag = $etag
    BodyLength = $document.RawContentLength
}

# A matching GET returns 304 without downloading the document again.
# SkipHttpErrorCheck lets PowerShell expose the expected non-2xx response.
$get = Invoke-WebRequest $documentUrl -Headers @{ 'If-None-Match' = $etag } -SkipHttpErrorCheck
[pscustomobject]@{
    StatusCode = $get.StatusCode
    ETag = $get.Headers['ETag'][0]
    BodyLength = $get.RawContentLength
}

# HEAD revalidates the same tag and also returns 304 with no body.
$head = Invoke-WebRequest $documentUrl -Method Head -Headers @{ 'If-None-Match' = $etag } -SkipHttpErrorCheck
[pscustomobject]@{
    StatusCode = $head.StatusCode
    ETag = $head.Headers['ETag'][0]
    BodyLength = $head.RawContentLength
}

# An outdated tag returns 200 with the current document and its current ETag.
$stale = Invoke-WebRequest $documentUrl -Headers @{ 'If-None-Match' = '"outdated-document"' }
[pscustomobject]@{
    StatusCode = $stale.StatusCode
    ETag = $stale.Headers['ETag'][0]
    BodyLength = $stale.RawContentLength
}
```

The four rows have status codes 200, 304, 304, and 200. All show the same
ETag. The first and last responses contain the document; the middle two have
body length 0. Keep your cached document when revalidation returns 304.
After a 200 response, replace the cached document and tag with the returned
values. These requests do not change albums or other server state.

The ETag is derived from the embedded document's content, so restarting an
unchanged binary keeps the tag valid. A binary with changed OpenAPI content
returns a different tag. See [API docs](../README.md#api-docs) for the
interactive documentation.
