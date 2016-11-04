package factoid

import "github.com/riking/homeapi/marvin"

func RunFactoid(info FactoidInfo, source marvin.ActionSource, args []string) (string, error) {
	// TODO lmao
	return info.RawSource, nil
}
