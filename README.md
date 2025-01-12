# Remote Image Extension for Godot 4

**This extension is WIP and unstable and subject to changes**

## Setup and build

Follow the installation for [graphics.gd](https://github.com/grow-graphics/gd) and you should be able to simply run `gd` to build and start the project. This repo comes with a proof of concept godot project. The aim is to at some point bundle the resulting extension into the `RemoteImage` addon that is setup inside the proof of concept project.\
See goals below.

## What is this actually?

The extension provides a new resource `RemoteImageTexture`.
If the `RemoteImage` addon is enabled a `RemoteImageLoader` node is added to the projects autoload nodes as `RILoader`. Every `RemoteImageTexture` instance requires this autoload to exist as it will ping the loader to load the URL and update the image after loading is complete.
Currently the loader tries to get the image format from the HTTP `Content-Type` response header or if that doesnt contain anything useful it falls back to looking at path of the URL and tries to determine the format based on the file extension.

The loader supports:

* jpeg
* png
* bmp
* webp
* tiff ( via conversion to jpeg )

## Goals

* gif support ( well ... it would be a lot smarter to expose a different resource - like https://github.com/BOTLANNER/godot-gif already does - with somethign based on AnimatedTexture but it also would be nice to not care and just have one remote resource that "just works" with any URL so maybe everything is based on AnimatedTexture instead of ImageTexture? Idk )
* CI building for all available OS and bundling of `RemoteImage` addon structure
* Read magic bits as primary method to identify image file format