@tool
class_name RemoteImage extends EditorPlugin

func _init() -> void:
	add_autoload_singleton("RILoader", "res://addons/RemoteImage/RILoader.gd")
	
	if !ProjectSettings.has_setting("RemoteImage/General/FallbackResourcePath"):
		ProjectSettings.set_setting("RemoteImage/General/FallbackResourcePath", "res://addons/RemoteImage/download.png")
		ProjectSettings.set_initial_value("RemoteImage/General/FallbackResourcePath", "res://addons/RemoteImage/download.png")
 
