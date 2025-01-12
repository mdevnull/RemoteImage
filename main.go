// This file is all you need to start a project.
// Save it somewhere, install the `gd` command and use `gd run` to get started.
package main

import (
	"graphics.gd/classdb"
	"graphics.gd/startup"
)

func main() {
	classdb.Register[RemoteImageLoader]()
	classdb.Register[RemoteImageTexture]()
	startup.Engine()
}
