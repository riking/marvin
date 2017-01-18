package lualib

import "github.com/dariusk/corpora"

// This function works around a bug in Gogland 1.0 EAP
func corporaAsset(path string) ([]byte, error) {
	return corpora.Asset(path)
}

func corporaAssetDir(name string) ([]string, error) {
	return corpora.AssetDir(name)
}
