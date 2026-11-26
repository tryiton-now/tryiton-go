# TryItOn Go SDK — AI Virtual Try-On API

Official Go client for the [TryItOn](https://tryiton.now) virtual try-on API. Add photoreal AI virtual try-on for clothing, accessories, hairstyles, and tattoos to your Go application with a few lines of code.

- Virtual clothing try-on and accessory try-on (eyewear, footwear, headwear, jewelry)
- Hairstyle and tattoo try-on
- Standard library only (no dependencies), context-aware, with a built-in job polling helper

Full API reference: [docs.tryiton.now](https://docs.tryiton.now) · Get an API key: [tryiton.now/app/developer](https://tryiton.now/app/developer)

## Installation

```bash
go get github.com/tryiton-now/tryiton-go/v2
```

Requires Go 1.20 or later.

## Quickstart: run a virtual try-on

Submit a garment and a model photo, then wait for the generated result image.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	tryiton "github.com/tryiton-now/tryiton-go/v2"
)

func main() {
	client, err := tryiton.New(os.Getenv("TRYITON_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Submit a clothing try-on
	jobID, err := client.TryOnFashion(ctx, tryiton.FashionParams{
		ModelImage:   "https://example.com/model.jpg",
		GarmentImage: "https://example.com/tshirt.jpg",
		Category:     "clothing",
		Subcategory:  "tops",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Poll until the job completes and return the output image URL(s)
	urls, err := client.WaitForResult(ctx, jobID, 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(urls[0]) // CDN URL, available for 72 hours
}
```

Image inputs accept a public URL or a base64 data URL (`data:image/png;base64,...`).

## Core parameters

`TryOnFashion` covers clothing and accessory try-on. The most important fields of `FashionParams`:

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `ModelImage` | string | Yes | URL or base64 data URL of the person. |
| `GarmentImage` | string | Yes | URL or base64 data URL of the garment or accessory. |
| `Category` | string | No | Item type: `auto`, `clothing`, `eyewear`, `footwear`, `headwear`, `jewelry`, `accessories`, or `others`. `auto` detects it for you. |
| `Subcategory` | string | No | Required for `clothing` (`tops`, `bottoms`, `dresses`), `jewelry`, and `accessories`. |

Additional options (`Mode` and `ModerationLevel` for clothing; `NumSamples` 1–4 and `OutputFormat` `png`/`jpeg` for every try-on, including hairstyle and tattoo) are documented in the [API reference](https://docs.tryiton.now).

## Other endpoints

```go
// Hairstyle try-on (see tryiton.Haircuts for all supported values)
client.TryOnHairstyle(ctx, tryiton.HairstyleParams{FaceImage: faceURL, Haircut: "BuzzCut", HairColor: "ash blonde"})

// Tattoo try-on — place it with free text...
client.TryOnTattoo(ctx, tryiton.TattooParams{BodyImage: bodyURL, DesignImage: designURL, Placement: "on the right forearm, small"})
// ...or pin the exact spot with a region box (normalized 0–1, from the image's top-left)
client.TryOnTattoo(ctx, tryiton.TattooParams{BodyImage: bodyURL, DesignImage: designURL, Region: &tryiton.TattooRegion{X: 0.32, Y: 0.18, W: 0.28, H: 0.34}})

// Poll a job manually, or check your credit balance
status, _ := client.GetStatus(ctx, jobID)  // *Status{ Status, Output, Error }
credits, _ := client.GetCredits(ctx)        // *Credits{ OnDemand, Subscription, Purchased, Reserved }
```

## Live try-on (real-time webcam)

Your server prepares garments and starts sessions with the API key. The shopper's browser streams with the
[`tryiton-live`](https://www.npmjs.com/package/tryiton-live) package, using only the session-scoped response.

```go
// Once per product, e.g. at catalog sync (cached per image + category):
garment, err := client.PrepareLiveGarment(ctx, tryiton.PrepareLiveGarmentParams{
	GarmentImage: productImageURL,
	Category:     "clothing",
	Subcategory:  "tops",
})

// When a shopper opens live try-on. Return start to the browser as-is:
start, err := client.StartLiveSession(ctx, tryiton.StartLiveSessionParams{
	GarmentID:     garment.ID,
	EndUserID:     shopperID,                  // one stream per end user
	AllowedOrigin: "https://shop.example.com", // locks the media token to your site
})
json.NewEncoder(w).Encode(start) // the JSON tags reproduce the API response

client.GetLiveSession(ctx, start.Session.ID)  // *LiveSession{ Status, Seconds, CreditsCharged, ... }
client.StopLiveSession(ctx, start.Session.ID) // end it from your server
client.GetLiveConfig(ctx)                     // *LiveConfig{ Enabled, SecondsPerCredit, MaxSeconds, ... }
```

Billing is per `SecondsPerCredit` seconds of streamed video, from the first try-on frame. Garment switches during a
session are free.

## Error handling

API and job failures are returned as `*tryiton.Error`, which carries the HTTP status code and the API error name.

```go
urls, err := client.WaitForResult(ctx, jobID, 2*time.Second)
if err != nil {
	var apiErr *tryiton.Error
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Status, apiErr.Name, apiErr.Message) // e.g. 429, "OutOfCredits"
	}
}
```

Failed jobs come back from `WaitForResult` as an `*Error` with `Status` 0, `Name` `"ProcessingError"` and the job's
failure reason as `Message`.

## Notes

- Output image URLs expire 72 hours after completion. Download any results you want to keep.
- Failed jobs are never charged.

## Documentation

Full documentation, parameter reference, and guides: [docs.tryiton.now](https://docs.tryiton.now)

## Development

```bash
gofmt -l .        # must print nothing
go vet ./...
go test -race ./...
```

`testdata/sdk-cases.json` is the contract fixture shared by every TryItOn SDK. It's generated, so don't edit it here:
change `tryiton/backend/api-contract/sdk-cases.json` and run `node tryiton-sdk/sync-contract.mjs`. `contract_test.go`
runs every case against a local `httptest` server. Releases are git tags (`vX.Y.Z`); bump `Version` in `version.go`
to match and add the entry to [CHANGELOG.md](CHANGELOG.md). CI runs the checks above on Go 1.20, 1.21 and 1.22.

## License

MIT
