package mcping

import (
	"regexp"
)

//PingResponse contains all known fields of the ping response packet
type PingResponse struct {
	Latency  uint           `json:"latency"`  //Latency in ms
	Online   int            `json:"online"`   //Amount of online players
	Max      int            `json:"max"`      //Maximum amount of players
	Protocol int            `json:"protocol"` //E.g '4'
	Favicon  []byte         `json:"favicon"`  //Base64 encoded favicon in data URI format
	Motd     string         `json:"motd"`
	Server   string         `json:"server"`  //E.g 'PaperSpigot'
	Version  string         `json:"version"` //E.g "1.7.10"
	Sample   []PlayerSample `json:"sample"`
}

//Code unknown to work, playing around
func stripMotd(motd string) string {
	return regexp.MustCompile("\\s+").ReplaceAllString(regexp.MustCompile("`+.").ReplaceAllString(
		regexp.MustCompile("[^\x20-\x7f]").ReplaceAllString(
			motd, "`"),
		""),
		" ")
}
