package imageclient

// DefaultPublicBase stands in when KUN_IMAGE_PUBLIC_BASE_URL is unset. In
// production the domain is set once in docker-compose.prod.yml, which feeds the
// API and the web from the same value.
const DefaultPublicBase = "https://image.kungal.iloveren.link"

// KnownHosts are the domains the shared NextMoe image service has been served
// under. The configured base's host is recognized on top of these at runtime;
// a host stays listed after the base moves on, because text written while it
// was current still names it and must keep folding into /image/<hash> tokens.
var KnownHosts = []string{
	"image.kungal.iloveren.link",
	"image.kungal.com",
}
