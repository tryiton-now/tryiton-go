# Changelog

All notable changes to the TryItOn Go SDK are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project follows
[Semantic Versioning](https://semver.org/). Versions are released as git tags (`vX.Y.Z`).

## [Unreleased]

### Deprecated

- `FashionParams.Mode` and `FashionParams.ModerationLevel`: the API ignores `mode` and validates `moderation_level` without applying it. Marked `Deprecated:`; removed in v3.

## [2.1.0] - 2026-11-26

### Added

- Live (real-time webcam) try-on endpoints: `GetLiveConfig`, `PrepareLiveGarment`, `StartLiveSession`,
  `GetLiveSession` and `StopLiveSession`, with their types (`LiveConfig`, `LiveGarment`, `LiveSession`,
  `LiveSessionStart`, ...). `json.Marshal` of a `LiveSessionStart` reproduces the API response for the browser.
- `Version` constant. Every request now sends `User-Agent: tryiton-go/<Version>`.
- `JobError` implements `json.Unmarshaler` and `json.Marshaler` (see Fixed).
- Contract tests against the fixture shared by all TryItOn SDKs (`testdata/sdk-cases.json`), and CI running
  `gofmt`, `go vet` and `go test -race` on Go 1.20, 1.21 and 1.22.

### Fixed

- API errors now carry the envelope's `error.name` and `error.message`. Before, `Error.Name` was `""` and
  `Error.Message` was `"HTTP 4xx"` for every API error. The older top-level `error`/`message` strings are still read.
- `GetStatus` and `WaitForResult` no longer fail with a `DecodeError` on a failed job. `GET /status` returns the
  failure reason as a plain string; it now decodes to `JobError{Name: "ProcessingError", Message: <reason>}`, and
  `WaitForResult` returns an `*Error` with `Status` 0, `Name` `"ProcessingError"` and the reason as `Message`.
  A `JobError` named `"ProcessingError"` marshals back to that plain string, so a `Status` re-marshals to the API shape.

### Changed

- The code is split from `tryiton.go` into `client.go`, `errors.go`, `tryon.go`, `status.go`, `live.go` and
  `version.go`. No API change.

## [2.0.0] - 2026-07-14

### Changed

- **Breaking:** `TryOnClothes` / `ClothesParams` are renamed to `TryOnFashion` / `FashionParams`, and the request goes
  to `POST /tryon/fashion` instead of `/tryon/clothes`. The module path is now `github.com/tryiton-now/tryiton-go/v2`.

## [1.1.0] - 2026-07-14

### Added

- `TattooParams.Region` (`TattooRegion`) to pin a tattoo to an exact spot on the body image.

## [1.0.0] - 2026-06-24

### Changed

- **Breaking:** removed the `seed`, `segmentation_free` and `garment_photo_type` clothing parameters.
- `NumSamples` (1-4) and `OutputFormat` (`png`/`jpeg`) apply to clothing, hairstyle and tattoo try-on.
  `ModerationLevel` is documented with `conservative`, `permissive` and `none`.

## [0.1.0] - 2026-06-22

### Added

- Initial release: clothing, hairstyle and tattoo try-on, `GetStatus`, `WaitForResult` and `GetCredits`.

[Unreleased]: https://github.com/tryiton-now/tryiton-go/compare/v2.1.0...HEAD
[2.1.0]: https://github.com/tryiton-now/tryiton-go/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/tryiton-now/tryiton-go/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/tryiton-now/tryiton-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/tryiton-now/tryiton-go/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/tryiton-now/tryiton-go/releases/tag/v0.1.0
