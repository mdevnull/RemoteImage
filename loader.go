package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/image/tiff"
	"graphics.gd/classdb"
	"graphics.gd/classdb/Image"
	"graphics.gd/classdb/Node"
	"graphics.gd/variant/Dictionary"
	"graphics.gd/variant/Float"
)

type imageFormat int

const (
	formatUnknown imageFormat = iota
	formatPNG
	formatJPEG
	formatBMP
	formatWEBP
	formatTIFF
	formatSVG
	formatTGA
)

// RemoteImageLoader is made available as the autoload node `RILoader`
type RemoteImageLoader struct {
	classdb.Extension[RemoteImageLoader, Node.Instance] `gd:"RemoteImageLoader"`

	LoadingImages Dictionary.Map[int, *RemoteImageTexture] `gd:"loading_images"`

	nextLoadCnt         int
	loadedImageDataLock sync.Mutex
	loadedImageData     map[int]imageResult
}

type imageResult struct {
	data   []byte
	format imageFormat
}

func (ril *RemoteImageLoader) IsSingleton() {}

func (ril *RemoteImageLoader) Ready() {
	ril.LoadingImages = Dictionary.New[int, *RemoteImageTexture]()
	ril.loadedImageData = make(map[int]imageResult)
	ril.loadedImageDataLock = sync.Mutex{}
}

func (ril *RemoteImageLoader) Process(delta Float.X) {
	if len(ril.loadedImageData) <= 0 {
		return
	}

	if !ril.loadedImageDataLock.TryLock() {
		return
	}
	defer ril.loadedImageDataLock.Unlock()

	for resourceUID, result := range ril.loadedImageData {
		remoteImageIndex := ril.LoadingImages.Index(resourceUID)
		imgResource := Image.New()

		var err error = nil
		switch result.format {
		case formatUnknown:
			panic("unknown image format")
		case formatJPEG:
			err = imgResource.LoadJpgFromBuffer(result.data)
		case formatPNG:
			err = imgResource.LoadPngFromBuffer(result.data)
		case formatWEBP:
			err = imgResource.LoadWebpFromBuffer(result.data)
		case formatBMP:
			err = imgResource.LoadBmpFromBuffer(result.data)
		case formatSVG:
			err = imgResource.LoadSvgFromBuffer(result.data)
		case formatTGA:
			err = imgResource.LoadTgaFromBuffer(result.data)
		}

		if err != nil {
			fmt.Println(err)
			imgResource.AsRefCounted()[0].Unreference()
			continue
		}

		remoteImageIndex.Super().SetImage(imgResource)
	}

	for k := range ril.loadedImageData {
		delete(ril.loadedImageData, k)
	}
}

func (ril *RemoteImageLoader) LoadRemoteImage(remoteImageRes *RemoteImageTexture) {
	ril.LoadingImages.SetIndex(ril.nextLoadCnt, remoteImageRes)

	go ril.loadRemoteImage(
		remoteImageRes.URL,
		ril.nextLoadCnt,
	)

	ril.nextLoadCnt += 1
}

func (ril *RemoteImageLoader) loadRemoteImage(url string, uid int) {
	if url == "" {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("go dl error: %s\n", err.Error()))
	}

	if resp.StatusCode > 299 {
		panic(fmt.Sprintf("go dl error: returned non 2XX code %d\n", resp.StatusCode))
	}

	imgBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		panic(fmt.Sprintf("go read error: %s\n", err.Error()))
	}

	if len(imgBytes) <= 0 {
		panic("go read error: empty response\n")
	}

	format := getTypeFromHeader(resp.Header.Get("content-type"))
	if format == formatUnknown {
		format = getTypeFromExtension(url)
	}

	if format == formatTIFF {
		switch format {
		case formatTIFF:
			format = formatJPEG
			imgBytes, err = convertTiff(imgBytes)
			if err != nil {
				panic(err)
			}
		}
	}

	ril.loadedImageDataLock.Lock()
	defer ril.loadedImageDataLock.Unlock()
	ril.loadedImageData[uid] = imageResult{imgBytes, format}
}

func getTypeFromExtension(uri string) imageFormat {
	urlObj, err := url.Parse(uri)
	if err != nil {
		fmt.Println("unable to parse URL")
		return formatUnknown
	}

	lastDotIndex := strings.LastIndex(urlObj.Path, ".")
	if lastDotIndex == -1 {
		fmt.Println("no dot in path")
		return formatUnknown
	}
	extension := urlObj.Path[lastDotIndex+1:]
	switch strings.ToLower(extension) {
	case "png":
		return formatPNG
	case "jpg":
		return formatJPEG
	case "jpeg":
		return formatJPEG
	case "webp":
		return formatWEBP
	case "tif":
		return formatTIFF
	case "tiff":
		return formatTIFF
	case "bmp":
		return formatBMP
	case "svg":
		return formatSVG
	case "tga":
		return formatTGA
	case "tpic":
		return formatTGA
	default:
		return formatUnknown
	}
}

func getTypeFromHeader(headerStr string) imageFormat {
	switch strings.ToLower(headerStr) {
	case "image/jpeg":
		return formatJPEG
	case "image/png":
		return formatPNG
	case "image/bmp":
		return formatBMP
	case "image/webp":
		return formatWEBP
	case "image/tiff":
		return formatTIFF
	case "image/svg+xml":
		return formatSVG
	case "image/svg":
		return formatSVG
	case "image/x-tga":
		return formatTGA
	case "image/x-targa":
		return formatTGA
	default:
		return formatUnknown
	}
}

func convertTiff(tiffData []byte) ([]byte, error) {
	img, err := tiff.Decode(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("error reading tiff: %w", err)
	}

	jpegBuffer := &bytes.Buffer{}
	if err := jpeg.Encode(jpegBuffer, img, &jpeg.Options{}); err != nil {
		return nil, fmt.Errorf("error encoding jpeg: %w", err)
	}

	return jpegBuffer.Bytes(), nil
}
