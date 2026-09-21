# Structured RPC errors

Return a structured error to preserve its status across service boundaries:

```go
return &errors.Error{
    Id: "media", Code: errors.CodeFailedPrecondition,
    Detail: "media is not ready", Reason: "MEDIA_NOT_READY", Domain: "media",
}
```

Use standard-library `errors.Is` and `errors.As` after transport decoding:

```go
if stderrors.Is(err, microerrors.NotFound("", "")) {
    // Handle a downstream 404 without turning it into a 500.
}
var detail *microerrors.Error
if stderrors.As(err, &detail) {
    // detail.Reason and detail.Domain are stable application identifiers.
}
```

Here `stderrors` is the standard `errors` package and `microerrors` is
`go-micro.dev/v6/errors`. `Is` compares the status code and any nonempty reason
or domain on the target; IDs and human-readable detail strings do not need to
match. Use a target such as `&microerrors.Error{Code: 412, Reason:
"MEDIA_NOT_READY", Domain: "media"}` for a narrower match. Zero-code errors do
not match by status. Applications choose their own reason/domain vocabulary.

`microerrors.As(err)` already returns `(*Error, bool)` for wrapped structured
errors. `FromError` retains its existing single-result signature for backwards
compatibility and now unwraps structured errors before falling back to parsing.

## Status translation

The default RPC transport preserves the structured error. Native gRPC uses this
table; the structured protobuf detail retains reason/domain when both ends use
Go Micro. A gRPC client also tolerates unrelated status details.

| Constructor | HTTP code | gRPC code |
| --- | --- | --- |
| BadRequest | 400 | InvalidArgument |
| Unauthorized | 401 | Unauthenticated |
| Forbidden | 403 | PermissionDenied |
| NotFound | 404 | NotFound |
| MethodNotAllowed | 405 | Unknown |
| Timeout | 408 | DeadlineExceeded |
| Conflict / AlreadyExists | 409 | AlreadyExists |
| FailedPrecondition | 412 | FailedPrecondition |
| ResourceExhausted | 429 | ResourceExhausted |
| InternalServerError | 500 | Internal |
| New(..., 501) | 501 | Unimplemented |
| Unavailable | 503 | Unavailable |

## Server diagnostics

Install optional middleware on the server:

```go
server.WrapHandler(wrapper.LogRawErrors(log))
```

`wrapper` is `go-micro.dev/v6/wrapper`. It logs the service, endpoint, whether the
error is structured, and the raw failure returned by the handler before
transport translation. It returns the error unchanged and never logs request
payloads. Raw errors can contain sensitive application details, so choose the
log sink deliberately. If a handler masks an error itself, record the original
there: middleware cannot recover information already discarded by the handler.
