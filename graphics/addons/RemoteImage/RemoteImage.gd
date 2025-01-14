@tool
class_name RemoteImage extends EditorPlugin

func _init() -> void:
	add_autoload_singleton("RILoader", "res://addons/RemoteImage/RILoader.gd")
	
	if !ProjectSettings.has_setting("RemoteImage/General/FallbackResourcePath"):
		ProjectSettings.set_setting("RemoteImage/General/FallbackResourcePath", "res://addons/RemoteImage/download.png")
	
	ProjectSettings.set_initial_value("RemoteImage/General/FallbackResourcePath", "res://addons/RemoteImage/download.png")
	
	if !ProjectSettings.has_setting("RemoteImage/General/Cache"):
		ProjectSettings.set_setting("RemoteImage/General/Cache", 0)
	
	ProjectSettings.set_initial_value("RemoteImage/General/Cache", 0)
	ProjectSettings.add_property_info({
		"name": "RemoteImage/General/Cache",
		"type": TYPE_INT,
		"hint": PROPERTY_HINT_ENUM,
		"hint_string": "None,Memory,File"
	})
