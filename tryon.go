package tryiton

import (
	"context"
	"net/http"
)

// Haircuts lists the supported `haircut` values for hairstyle try-on.
var Haircuts = []string{
	"Afro", "BobCut", "BowlCut", "BoxBraids", "BuzzCut", "Chignon", "CombOver",
	"CornrowBraids", "CurlyBob", "CurlyShag", "DoubleBun", "Dreadlocks", "FauxHawk",
	"FishtailBraid", "LongCurly", "LongHairTiedUp", "LongHimeCut", "LongStraight",
	"LongTwintails", "LongWavy", "LongWavyCurtainBangs", "ManBun", "MessyTousled",
	"PixieCut", "Pompadour", "Ponytail", "ShortCurlyPixie", "ShortTwintails",
	"ShoulderLengthHair", "Spiky", "TexturedFringe", "TwinBraids", "Updo", "WavyShag",
}

// FashionParams are the inputs for a clothing/accessory try-on. Image fields
// accept a public URL or a base64 data URL. Category is one of: auto, clothing,
// eyewear, footwear, headwear, jewelry, accessories, others. clothing, jewelry,
// and accessories require a Subcategory.
// NumSamples is the number of output images (1-4); charged per image.
// OutputFormat is "png" or "jpeg" (defaults to "png"). ModerationLevel is one of
// "conservative", "permissive", or "none".
type FashionParams struct {
	ModelImage      string `json:"model_image"`
	GarmentImage    string `json:"garment_image"`
	Category        string `json:"category,omitempty"`
	Subcategory     string `json:"subcategory,omitempty"`
	Mode            string `json:"mode,omitempty"`
	NumSamples      int    `json:"num_samples,omitempty"`
	OutputFormat    string `json:"output_format,omitempty"`
	ModerationLevel string `json:"moderation_level,omitempty"`
}

// HairstyleParams are the inputs for a hairstyle try-on. NumSamples is the
// number of output images (1-4); OutputFormat is "png" or "jpeg".
type HairstyleParams struct {
	FaceImage    string `json:"face_image"`
	Haircut      string `json:"haircut"`
	HairColor    string `json:"hair_color,omitempty"`
	NumSamples   int    `json:"num_samples,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// TattooRegion is a rectangle on the body image, normalized 0-1 from its
// TOP-LEFT corner. Each side must be at least 0.06 of the image.
type TattooRegion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// TattooParams are the inputs for a tattoo try-on. NumSamples is the number of
// output images (1-4); OutputFormat is "png" or "jpeg".
//
// Placement is free text ("on the right forearm, small"); Region pins the exact
// spot, and with it set Placement only describes size/style.
type TattooParams struct {
	BodyImage    string        `json:"body_image"`
	DesignImage  string        `json:"design_image"`
	Placement    string        `json:"placement,omitempty"`
	Region       *TattooRegion `json:"region,omitempty"`
	NumSamples   int           `json:"num_samples,omitempty"`
	OutputFormat string        `json:"output_format,omitempty"`
}

// TryOnFashion puts a garment or accessory on a person and returns the job id.
func (c *Client) TryOnFashion(ctx context.Context, p FashionParams) (string, error) {
	return c.submit(ctx, "/tryon/fashion", p)
}

// TryOnHairstyle restyles a person's hair and returns the job id.
func (c *Client) TryOnHairstyle(ctx context.Context, p HairstyleParams) (string, error) {
	return c.submit(ctx, "/tryon/hairstyle", p)
}

// TryOnTattoo inks a design onto skin and returns the job id.
func (c *Client) TryOnTattoo(ctx context.Context, p TattooParams) (string, error) {
	return c.submit(ctx, "/tryon/tattoo", p)
}

func (c *Client) submit(ctx context.Context, path string, body any) (string, error) {
	var out struct {
		JobID string `json:"jobId"`
	}
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return "", err
	}
	return out.JobID, nil
}
