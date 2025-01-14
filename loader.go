package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/tiff"
	"graphics.gd/classdb"
	"graphics.gd/classdb/Image"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/OS"
	"graphics.gd/classdb/ProjectSettings"
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

	cache remoteImageCache
}

type imageResult struct {
	err    error
	data   []byte
	format imageFormat
}

func (ril *RemoteImageLoader) IsSingleton() {}

func (ril *RemoteImageLoader) Ready() {
	ril.LoadingImages = Dictionary.New[int, *RemoteImageTexture]()
	ril.loadedImageData = make(map[int]imageResult)
	ril.loadedImageDataLock = sync.Mutex{}

	switch getCacheSetting() {
	case cacheSettingMemory:
		ril.cache = newMemoryCache()
	case cacheSettingFilsystem:
		basePath := path.Join(OS.GetUserDataDir(), "RemoteImages")
		_, err := os.Stat(basePath)
		if err != nil {
			if err := os.Mkdir(basePath, os.ModePerm); err != nil {
				panic(fmt.Errorf("error creating cache base dir: %w", err))
			}
		}
		ril.cache = &fileCache{
			basePath: basePath,
		}
	default:
		ril.cache = &noopCache{}
	}
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
		remoteImage := ril.LoadingImages.Index(resourceUID)

		if result.err != nil {
			remoteImage.emitError(result.err)
			continue
		}

		imgResource := Image.New()

		var err error = nil
		switch result.format {
		case formatUnknown:
			remoteImage.emitError(fmt.Errorf("error detecting image format"))
			continue
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
			// remoteImage.emitError(fmt.Errorf("error loading from buffer ( format: %d ): %w", result.format, err))
			// imgResource.AsRefCounted()[0].Unreference()
			// continue
		}

		remoteImage.Super().SetImage(imgResource)
	}

	for k := range ril.loadedImageData {
		delete(ril.loadedImageData, k)
	}
}

func (ril *RemoteImageLoader) LoadRemoteImage(remoteImageTexture *RemoteImageTexture) {
	ril.LoadingImages.SetIndex(ril.nextLoadCnt, remoteImageTexture)

	cacheEntry, err := ril.cache.get(remoteImageTexture.URL)
	if err != nil && err != errCacheMiss {
		fmt.Printf("cache error: %s\n", err.Error())
	}
	if err == nil {
		ril.setLoadResult(ril.nextLoadCnt, cacheEntry)
	} else {
		go ril.loadRemoteImage(
			remoteImageTexture.URL,
			ril.nextLoadCnt,
		)
	}

	ril.nextLoadCnt += 1
}

func (ril *RemoteImageLoader) loadRemoteImage(url string, uid int) {
	if url == "" {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		ril.setLoadResult(uid, imageResult{fmt.Errorf("download error: %w", err), nil, formatUnknown})
	}

	if resp.StatusCode > 299 {
		ril.setLoadResult(uid, imageResult{fmt.Errorf("returned non 2XX code %d", resp.StatusCode), nil, formatUnknown})
		return
	}

	imgBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		ril.setLoadResult(uid, imageResult{fmt.Errorf("read error: %w", err), nil, formatUnknown})
		return
	}

	if len(imgBytes) <= 0 {
		ril.setLoadResult(uid, imageResult{fmt.Errorf("empty server response"), nil, formatUnknown})
		return
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
				ril.setLoadResult(uid, imageResult{fmt.Errorf("error converting tiff: %w", err), nil, formatUnknown})
				return
			}
		}
	}

	imageRes := imageResult{nil, imgBytes, format}
	err = ril.cache.set(url, imageRes)
	if err != nil {
		fmt.Printf("cache write error: %s", err.Error())
	}

	ril.setLoadResult(uid, imageRes)
}

func (ril *RemoteImageLoader) setLoadResult(uid int, res imageResult) {
	ril.loadedImageDataLock.Lock()
	defer ril.loadedImageDataLock.Unlock()
	ril.loadedImageData[uid] = res
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

const (
	cacheSettingNoop int = iota
	cacheSettingMemory
	cacheSettingFilsystem
)

func getCacheSetting() int {
	settingVal := ProjectSettings.GetSetting("RemoteImage/General/Cache")
	var cacheOption int
	if settingVal != nil {
		cacheOption, _ = strconv.Atoi(fmt.Sprintf("%v", settingVal))
	}
	return cacheOption
}
