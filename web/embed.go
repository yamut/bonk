package web

import "embed"

//go:embed index.html style.css game.js
var Assets embed.FS
