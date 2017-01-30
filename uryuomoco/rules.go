package uryuomoco

var common = map[string]string{
	"a": "u",
	"u": "a",

	"b": "v",
	"v": "b",

	"c": "s",
	"s": "c",

	"d": "j",
	"j": "d",

	"e": "o",
	"o": "e",

	"f": "h",
	"h": "f",

	"g": "t",
	"t": "g",

	"k": "p",
	"p": "k",

	"l": "r",
	"r": "l",

	"m": "n",
	"n": "m",

	"qu": "w",
	"w":  "qu",

	"x": "z",
	"z": "x",
}

var englishOnly = map[string][]string{
	"i":   {"yu", "y"},
	"th":  {"ch"},
	"ch":  {"se"},
	"wh":  {"quo"},
	"ing": {"ot"},
	"gr":  {"tul"},
	"ss":  {"ais"},
	"ll":  {"ra"},
	"sh":  {"us"},
}

var uryuOnly = map[string][]string{
	"i":   {"y"},
	"yu":  {"i"},
	"ch":  {"th"},
	"se":  {"ch"},
	"quo": {"wh"},
	"ot":  {"ing"},
	"tul": {"gr"},
	"ais": {"ss"},
	"ra":  {"ll"},
	"us":  {"sh"},
}

// A StemRule is applied when a Uryuomoco word ends in a certain letter.
// When applied, the AddRune is added on the end of the word.
//
// This results in ambiguous Uryu -> English translations, so a dictionary is used to resolve those situations.
type StemRule struct {
	EndRune rune
	AddRune rune
}

var stemmingRules = []StemRule{
	{EndRune: 'e', AddRune: 'h'},
	{EndRune: 'E', AddRune: 'H'},
	{EndRune: 'j', AddRune: 'a'},
	{EndRune: 'J', AddRune: 'A'},
}
