package image

import (
	"github.com/m3o/m3o-go/client"
)

func NewImageService(token string) *ImageService {
	return &ImageService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type ImageService struct {
	client *client.Client
}

// Convert an image from one format (jpeg, png etc.) to an other either on the fly (from base64 to base64),
// or by uploading the conversion result.
func (t *ImageService) Convert(request *ConvertRequest) (*ConvertResponse, error) {
	rsp := &ConvertResponse{}
	return rsp, t.client.Call("image", "Convert", request, rsp)
}

// Resize an image on the fly without storing it (by sending and receiving a base64 encoded image), or resize and upload depending on parameters.
// If one of width or height is 0, the image aspect ratio is preserved.
// Optional cropping.
func (t *ImageService) Resize(request *ResizeRequest) (*ResizeResponse, error) {
	rsp := &ResizeResponse{}
	return rsp, t.client.Call("image", "Resize", request, rsp)
}

// Upload an image by either sending a base64 encoded image to this endpoint or a URL.
// To resize an image before uploading, see the Resize endpoint.
func (t *ImageService) Upload(request *UploadRequest) (*UploadResponse, error) {
	rsp := &UploadResponse{}
	return rsp, t.client.Call("image", "Upload", request, rsp)
}

type ConvertRequest struct {
	// base64 encoded image to resize,
	// ie. "data:image/png;base64, iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg=="
	Base64 string `json:"base64"`
	// output name of the image including extension, ie. "cat.png"
	Name string `json:"name"`
	// make output a URL and not a base64 response
	OutputUrl bool `json:"outputUrl"`
	// url of the image to resize
	Url string `json:"url"`
}

type ConvertResponse struct {
	Base64 string `json:"base64"`
	Url    string `json:"url"`
}

type CropOptions struct {
	// Crop anchor point: "top", "top left", "top right",
	// "left", "center", "right"
	// "bottom left", "bottom", "bottom right".
	// Optional. Defaults to center.
	Anchor string `json:"anchor"`
	// height to crop to
	Height int32 `json:"height"`
	// width to crop to
	Width int32 `json:"width"`
}

type Point struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type Rectangle struct {
	Max *Point `json:"max"`
	Min *Point `json:"min"`
}

type ResizeRequest struct {
	// base64 encoded image to resize,
	// ie. "data:image/png;base64, iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg=="
	Base64 string `json:"base64"`
	// optional crop options
	// if provided, after resize, the image
	// will be cropped
	CropOptions *CropOptions `json:"cropOptions"`
	Height      int64        `json:"height,string"`
	// output name of the image including extension, ie. "cat.png"
	Name string `json:"name"`
	// make output a URL and not a base64 response
	OutputUrl bool `json:"outputUrl"`
	// url of the image to resize
	Url   string `json:"url"`
	Width int64  `json:"width,string"`
}

type ResizeResponse struct {
	Base64 string `json:"base64"`
	Url    string `json:"url"`
}

type UploadRequest struct {
	// Base64 encoded image to upload,
	// ie. "data:image/png;base64, iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg=="
	Base64 string `json:"base64"`
	// Output name of the image including extension, ie. "cat.png"
	Name string `json:"name"`
	// URL of the image to upload
	Url string `json:"url"`
}

type UploadResponse struct {
	Url string `json:"url"`
}
