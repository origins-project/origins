package browse

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// file serves one of a record's attached files, or with a region the part of it the
// region marks.
//
// The path is not sanitised and then trusted; it is looked up among the files that were
// actually found beside the record when the corpus was read. A name that escapes the
// corpus and a name that was never attached fail the same test, so there is one rule
// to get right rather than two.
func (s *Server) file(w http.ResponseWriter, r *http.Request) {
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	ref, err := model.ParseEntityRef(r.PathValue("kind") + "/" + r.PathValue("slug"))
	if err != nil || c.KindOf(ref.Slug) != ref.Kind {
		http.NotFound(w, r)
		return
	}
	path, err := c.Attachment(ref.Slug, r.PathValue("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if marked := r.URL.Query().Get("region"); marked != "" && crop(w, path, marked) {
		return
	}
	http.ServeFile(w, r, path)
}

// crop writes the part of a picture a region marks, and reports whether it could. A
// picture in a format the standard library does not read, or a region that marks
// nothing, is served whole rather than refused: it is still a picture of the right
// person, only not as close, and check is what says the region is wrong.
func crop(w http.ResponseWriter, path, marked string) bool {
	region, ok := parseRegion(marked)
	if !ok {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	picture, format, err := image.Decode(f)
	if err != nil {
		return false
	}
	part, ok := picture.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return false
	}
	b := picture.Bounds()
	at := func(from int, size int, fraction float64) int { return from + int(fraction*float64(size)) }
	marks := image.Rect(at(b.Min.X, b.Dx(), region.Left), at(b.Min.Y, b.Dy(), region.Top),
		at(b.Min.X, b.Dx(), region.Right), at(b.Min.Y, b.Dy(), region.Bottom))
	if marks.Empty() {
		return false
	}

	var out bytes.Buffer
	kind := "image/png"
	if format == "jpeg" {
		kind, err = "image/jpeg", jpeg.Encode(&out, part.SubImage(marks), &jpeg.Options{Quality: 90})
	} else {
		err = png.Encode(&out, part.SubImage(marks))
	}
	if err != nil {
		return false
	}
	w.Header().Set("Content-Type", kind)
	_, _ = w.Write(out.Bytes())
	return true
}

// parseRegion reads a region as a route writes it: four fractions of the picture,
// left, top, right and bottom, within it and the right way round.
func parseRegion(marked string) (model.Region, bool) {
	edges := strings.Split(marked, ",")
	if len(edges) != 4 {
		return model.Region{}, false
	}
	var fractions [4]float64
	for i, edge := range edges {
		f, err := strconv.ParseFloat(strings.TrimSpace(edge), 64)
		if err != nil || f < 0 || f > 1 {
			return model.Region{}, false
		}
		fractions[i] = f
	}
	r := model.Region{Left: fractions[0], Top: fractions[1], Right: fractions[2], Bottom: fractions[3]}
	return r, r.Left < r.Right && r.Top < r.Bottom
}
