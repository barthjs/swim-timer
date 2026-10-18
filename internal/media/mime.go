package media

import "mime"

// init registers the MIME types of common file extensions that minimal container images do not ship.
func init() {
	// Alpine and scratch images ship without a MIME type table
	_ = mime.AddExtensionType(".azw", "application/vnd.amazon.ebook")
	_ = mime.AddExtensionType(".csv", "text/csv")
	_ = mime.AddExtensionType(".doc", "application/msword")
	_ = mime.AddExtensionType(".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	_ = mime.AddExtensionType(".epub", "application/epub+zip")
	_ = mime.AddExtensionType(".json", "application/json")
	_ = mime.AddExtensionType(".jsonld", "application/ld+json")
	_ = mime.AddExtensionType(".odp", "application/vnd.oasis.opendocument.presentation")
	_ = mime.AddExtensionType(".ods", "application/vnd.oasis.opendocument.spreadsheet")
	_ = mime.AddExtensionType(".odt", "application/vnd.oasis.opendocument.text")
	_ = mime.AddExtensionType(".pdf", "application/pdf")
	_ = mime.AddExtensionType(".ppt", "application/vnd.ms-powerpoint")
	_ = mime.AddExtensionType(".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	_ = mime.AddExtensionType(".rtf", "application/rtf")
	_ = mime.AddExtensionType(".txt", "text/plain")
	_ = mime.AddExtensionType(".xls", "application/vnd.ms-excel")
	_ = mime.AddExtensionType(".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	_ = mime.AddExtensionType(".xml", "application/xml")

	// Images
	_ = mime.AddExtensionType(".avif", "image/avif")
	_ = mime.AddExtensionType(".bmp", "image/bmp")
	_ = mime.AddExtensionType(".gif", "image/gif")
	_ = mime.AddExtensionType(".ico", "image/vnd.microsoft.icon")
	_ = mime.AddExtensionType(".jpeg", "image/jpeg")
	_ = mime.AddExtensionType(".jpg", "image/jpeg")
	_ = mime.AddExtensionType(".png", "image/png")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
	_ = mime.AddExtensionType(".tif", "image/tiff")
	_ = mime.AddExtensionType(".tiff", "image/tiff")
	_ = mime.AddExtensionType(".webp", "image/webp")

	// Audio
	_ = mime.AddExtensionType(".aac", "audio/aac")
	_ = mime.AddExtensionType(".cda", "application/x-cdf")
	_ = mime.AddExtensionType(".mid", "audio/midi")
	_ = mime.AddExtensionType(".midi", "audio/midi")
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")
	_ = mime.AddExtensionType(".oga", "audio/ogg")
	_ = mime.AddExtensionType(".ogg", "audio/ogg")
	_ = mime.AddExtensionType(".opus", "audio/opus")
	_ = mime.AddExtensionType(".wav", "audio/wav")
	_ = mime.AddExtensionType(".weba", "audio/webm")

	// Video
	_ = mime.AddExtensionType(".3g2", "video/3gpp2")
	_ = mime.AddExtensionType(".3gp", "video/3gpp")
	_ = mime.AddExtensionType(".avi", "video/x-msvideo")
	_ = mime.AddExtensionType(".mp4", "video/mp4")
	_ = mime.AddExtensionType(".mpeg", "video/mpeg")
	_ = mime.AddExtensionType(".ogv", "video/ogg")
	_ = mime.AddExtensionType(".ts", "video/mp2t")
	_ = mime.AddExtensionType(".webm", "video/webm")

	// Web Resources & Code
	_ = mime.AddExtensionType(".css", "text/css")
	_ = mime.AddExtensionType(".csh", "application/x-csh")
	_ = mime.AddExtensionType(".htm", "text/html")
	_ = mime.AddExtensionType(".html", "text/html")
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".mjs", "text/javascript")
	_ = mime.AddExtensionType(".php", "application/x-httpd-php")
	_ = mime.AddExtensionType(".sh", "application/x-sh")
	_ = mime.AddExtensionType(".xhtml", "application/xhtml+xml")

	// Fonts
	_ = mime.AddExtensionType(".eot", "application/vnd.ms-fontobject")
	_ = mime.AddExtensionType(".otf", "font/otf")
	_ = mime.AddExtensionType(".ttf", "font/ttf")
	_ = mime.AddExtensionType(".woff", "font/woff")
	_ = mime.AddExtensionType(".woff2", "font/woff2")

	// Archives & Containers
	_ = mime.AddExtensionType(".7z", "application/x-7z-compressed")
	_ = mime.AddExtensionType(".arc", "application/x-freearc")
	_ = mime.AddExtensionType(".bz", "application/x-bzip")
	_ = mime.AddExtensionType(".bz2", "application/x-bzip2")
	_ = mime.AddExtensionType(".gz", "application/gzip")
	_ = mime.AddExtensionType(".jar", "application/java-archive")
	_ = mime.AddExtensionType(".rar", "application/vnd.rar")
	_ = mime.AddExtensionType(".tar", "application/x-tar")
	_ = mime.AddExtensionType(".zip", "application/zip")

	// Miscellaneous / Binary
	_ = mime.AddExtensionType(".abw", "application/x-abiword")
	_ = mime.AddExtensionType(".bin", "application/octet-stream")
	_ = mime.AddExtensionType(".ics", "text/calendar")
	_ = mime.AddExtensionType(".mpkg", "application/vnd.apple.installer+xml")
	_ = mime.AddExtensionType(".ogx", "application/ogg")
	_ = mime.AddExtensionType(".vsd", "application/vnd.visio")
	_ = mime.AddExtensionType(".xul", "application/vnd.mozilla.xul+xml")
}
