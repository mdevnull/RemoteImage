extends Button

@onready var txt_input: TextEdit = $"../TextEdit"
@onready var img_container: VBoxContainer = $"../../ScrollContainer/VBoxContainer"

func _on_pressed() -> void:
	var rows = txt_input.text.split("\n")
	for row in rows:
		var new_centering = CenterContainer.new()
		var new_rect = TextureRect.new()
		var remote_img = RemoteImageTexture.new()
		new_rect.texture = remote_img
		
		new_centering.add_child(new_rect)
		img_container.add_child(new_centering)
		
		remote_img.SetURL(row)
