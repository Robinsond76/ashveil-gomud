import sys
from PIL import Image
im = Image.open(sys.argv[1]).convert("RGBA")
s = int(sys.argv[3]) if len(sys.argv) > 3 else 4
bg = Image.new("RGBA", im.size, (128, 128, 128, 255))
bg.alpha_composite(im)
bg.resize((im.width * s, im.height * s), Image.NEAREST).save(sys.argv[2])
