package main

import (
	_ "embed"
	"image/color"
	"log"
	"strings"
	"syscall/js"

	"golang.org/x/tools/cover"

	"github.com/nikolaydubina/go-cover-treemap/covertreemap"
	"github.com/nikolaydubina/treemap"
	"github.com/nikolaydubina/treemap/render"
)

var grey = color.RGBA{128, 128, 128, 255}

type Renderer struct {
	w          float64 // svg width
	h          float64 // svg height
	marginBox  float64
	paddingBox float64
	padding    float64
	fileText   string
}

func (r *Renderer) OnWindowResize(this js.Value, args []js.Value) interface{} {
	windowWidth := js.Global().Get("innerWidth").Int()
	windowHeight := js.Global().Get("innerHeight").Int()

	document := js.Global().Get("document")
	outputContainer := document.Call("getElementById", "output-container")
	fileInput := document.Call("getElementById", "file-input")

	w := windowWidth
	h := windowHeight - (outputContainer.Get("offsetTop").Int() - fileInput.Get("offsetHeight").Int())

	if h <= 4 {
		return false
	}

	r.w = float64(w)
	r.h = float64(h)

	r.Render()
	return false
}

func (r *Renderer) OnFileDrop(this js.Value, args []js.Value) interface{} {
	event := args[0]
	event.Call("preventDefault")

	fileReader := js.Global().Get("FileReader").New()
	fileReader.Set("onload", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		r.fileText = e.Get("target").Get("result").String()
		r.Render()
		return nil
	}))

	file := event.Get("dataTransfer").Get("files").Index(0)
	fileReader.Call("readAsText", file)

	return false
}

func (r *Renderer) OnDragOver(this js.Value, args []js.Value) interface{} {
	document := js.Global().Get("document")
	document.Call("getElementById", "file-input").Set("className", "file-input-hover")
	return false
}

func (r *Renderer) OnDragEnd(this js.Value, args []js.Value) interface{} {
	document := js.Global().Get("document")
	document.Call("getElementById", "file-input").Set("className", "")
	return false
}

func (r *Renderer) Render() {
	if r.fileText == "" {
		return
	}

	profiles, err := cover.ParseProfilesFromReader(strings.NewReader(r.fileText))
	if err != nil {
		log.Fatal(err)
	}

	treemapBuilder := covertreemap.NewCoverageTreemapBuilder(true)
	tree, err := treemapBuilder.CoverageTreemapFromProfiles(profiles)
	if err != nil {
		log.Fatal(err)
	}

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)
	treemap.SetNamesFromPaths(tree)
	treemap.CollapseLongPaths(tree)

	heatImputer := treemap.WeightedHeatImputer{EmptyLeafHeat: 0.5}
	heatImputer.ImputeHeat(*tree)

	palette, ok := render.GetPalette("RdYlGn")
	if !ok {
		log.Fatalf("can not get palette")
	}
	uiBuilder := render.UITreeMapBuilder{
		Colorer:     render.HeatColorer{Palette: palette},
		BorderColor: grey,
	}
	spec := uiBuilder.NewUITreeMap(*tree, r.w, r.h, r.marginBox, r.paddingBox, r.padding)
	renderer := render.SVGRenderer{}

	img := renderer.Render(spec, r.w, r.h)

	document := js.Global().Get("document")
	document.Call("getElementById", "output-container").Set("innerHTML", string(img))
	document.Call("getElementById", "file-input").Get("style").Set("display", "none")
}

func main() {
	c := make(chan bool)
	renderer := Renderer{
		marginBox:  4,
		paddingBox: 4,
		padding:    16,
	}

	document := js.Global().Get("document")
	fileInput := document.Call("getElementById", "file-input")

	fileInput.Set("ondragover", js.FuncOf(renderer.OnDragOver))
	fileInput.Set("ondragend", js.FuncOf(renderer.OnDragEnd))
	fileInput.Set("ondragleave", js.FuncOf(renderer.OnDragEnd))
	fileInput.Set("ondrop", js.FuncOf(renderer.OnFileDrop))

	js.Global().Set("onresize", js.FuncOf(renderer.OnWindowResize))

	renderer.OnWindowResize(js.Value{}, nil)

	<-c
}
