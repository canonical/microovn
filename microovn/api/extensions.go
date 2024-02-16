package api

var extensions = []string{
	"microovn:custom_encapsulation_ip",
}

// Extensions returns the list of MicroOVN extensions.
func Extensions() []string {
	return extensions
}
