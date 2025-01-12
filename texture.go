package main

import (
	"fmt"

	"graphics.gd/classdb"
	"graphics.gd/classdb/Engine"
	"graphics.gd/classdb/ImageTexture"
	"graphics.gd/classdb/MainLoop"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/ProjectSettings"
	"graphics.gd/classdb/Resource"
	"graphics.gd/classdb/ResourceLoader"
	"graphics.gd/classdb/SceneTree"
	"graphics.gd/classdb/Texture2D"
	"graphics.gd/classdb/Window"
	"graphics.gd/variant/Object"
)

type RemoteImageTexture struct {
	classdb.Extension[RemoteImageTexture, ImageTexture.Instance] `gd:"RemoteImageTexture"`

	URL           string             `gd:"image_url"`
	FallbackImage Texture2D.Instance `gd:"fallback_image"`
}

func (ri *RemoteImageTexture) OnCreate() {
	if ri.FallbackImage != (Texture2D.Instance{}) {
		ri.Super().SetImage(ri.FallbackImage.GetImage())
	} else {
		settingVal := ProjectSettings.GetSetting("RemoteImage/General/FallbackResourcePath")
		fallbackPath := "res://addons/RemoteImage/download.png"
		if settingVal != nil {
			fallbackPath = fmt.Sprintf("%v", settingVal)
		}
		fallbackImageResource := Resource.Instance(ResourceLoader.Load(fallbackPath))
		fallbackTexture, ok := classdb.As[Texture2D.Instance](fallbackImageResource)
		if !ok {
			panic("Fallback resource is not a Texture2D")
		}
		ri.Super().SetImage(fallbackTexture.GetImage())
	}

	ri.onURLChange()
}

func (ri *RemoteImageTexture) SetURL(uri string) {
	ri.URL = uri
	ri.onURLChange()
}

func (ri *RemoteImageTexture) onURLChange() {
	if ri.URL != "" {
		if tree, ok := Object.Is[SceneTree.Instance](MainLoop.Instance(Engine.GetMainLoop())); ok {
			rilNode := Node.Instance(Window.Instance(tree.Root()).AsNode().GetNode("RILoader"))
			if riLoader, ok := classdb.As[*RemoteImageLoader](rilNode); ok {
				riLoader.LoadRemoteImage(ri)
			}
		}
	}
}
