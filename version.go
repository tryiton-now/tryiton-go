package tryiton

// Version is the SDK version, sent in the User-Agent header. Keep it in sync
// with the release tag (v2.2.0 → "2.2.0").
const Version = "2.2.0"

// userAgent identifies the SDK on every request, e.g. "tryiton-go/2.2.0".
const userAgent = "tryiton-go/" + Version
