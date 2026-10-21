"""Notocjk writes the CJK faces the charts are drawn in from Noto Sans CJK's collection:

    python tools/notocjk.py NotoSansCJK-Regular.ttc internal/chart/draw/fonts/NotoSansCJK-Regular.ttf
    python tools/notocjk.py NotoSansCJK-Medium.ttc internal/chart/draw/fonts/NotoSansCJK-Medium.ttf

It takes the collection's first face, whose character map is every face's and whose
glyphs are every region's, and rewrites its outlines as TrueType curves. The PDF writer
cuts a font down to the glyphs a chart uses, and cannot cut down a CFF font with more
than one dictionary, as Noto Sans CJK's are. It needs fontTools.
"""
import sys

from fontTools.pens.cu2quPen import Cu2QuPen
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont, newTable

source, out = sys.argv[1], sys.argv[2]
font = TTFont(source, fontNumber=0)
order = font.getGlyphOrder()
glyphs = font.getGlyphSet()
quadratic = {}
for name in order:
    pen = TTGlyphPen(None)
    glyphs[name].draw(Cu2QuPen(pen, max_err=1.0, reverse_direction=True))
    quadratic[name] = pen.glyph()

font["loca"] = newTable("loca")
glyf = font["glyf"] = newTable("glyf")
glyf.glyphOrder = order
glyf.glyphs = quadratic
del font["CFF "]
if "VORG" in font:
    del font["VORG"]
glyf.compile(font)

maxp = font["maxp"]
maxp.tableVersion = 0x00010000
for attr in ("maxZones", "maxTwilightPoints", "maxStorage", "maxFunctionDefs",
             "maxInstructionDefs", "maxStackElements", "maxSizeOfInstructions",
             "maxComponentElements", "maxComponentDepth"):
    setattr(maxp, attr, 0)
maxp.maxZones = 1
maxp.recalc(font)

post = font["post"]
post.formatType = 3.0
post.extraNames, post.mapping = [], {}
font["head"].glyphDataFormat = 0
font["head"].indexToLocFormat = 1
font.sfntVersion = "\x00\x01\x00\x00"
font.save(out)
