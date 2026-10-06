"""Contract tests for authored sheet integration; run with unittest discover."""
import tempfile
import unittest
from pathlib import Path

from PIL import Image

import authored
from palette import PAL


class AuthoredTests(unittest.TestCase):
    def test_missing_source_preserves_procedural_art(self):
        with tempfile.TemporaryDirectory() as tmp:
            image = Image.new("RGBA", (32, 32))
            self.assertIs(authored.replacement("missing.png", image, {}, Path(tmp)), image)

    def test_source_is_used_and_invalid_exports_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            generated = Image.new("RGBA", (32, 32))
            source = generated.copy()
            source.putpixel((16, 30), (*PAL["iron.m"], 255))
            meta = {"kind": "map-unit", "frame": [32, 32]}
            source.save(root / "unit.png")
            result = authored.replacement("unit.png", generated, meta, root)
            self.assertEqual(result.tobytes(), source.tobytes())
            for pixel in [(1, 2, 3, 255), (*PAL["iron.m"], 128)]:
                bad = source.copy()
                bad.putpixel((16, 30), pixel)
                bad.save(root / "unit.png")
                with self.assertRaises(ValueError):
                    authored.replacement("unit.png", generated, meta, root)
            for bad in [Image.new("RGBA", (64, 32)), generated]:
                bad.save(root / "unit.png")
                with self.assertRaises(ValueError):
                    authored.replacement("unit.png", generated, meta, root)


if __name__ == "__main__":
    unittest.main()
