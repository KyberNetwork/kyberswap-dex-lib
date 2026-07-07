package cloberob

import (
	_ "embed"
)

//go:embed abi/BookManager.json
var bookManagerBytes []byte

//go:embed abi/BookViewer.json
var bookViewerBytes []byte
