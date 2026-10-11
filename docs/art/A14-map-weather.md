# A14: Weather on the map

Read [00-standards.md](00-standards.md) first. Attach the approved A2
terrain tiles (a forest, a road and a water tile at least) and the A13
light markers, so the weather sits on the same ground.

The map is about to show the weather (living map phase LM4): rain falling
across the tiles, a storm with lightning, fog and mist drifting, cloud
shadows on an overcast day, and snow. **The game moves everything itself**
by sliding these textures across the map, so most files are a single still
texture that **tiles seamlessly** in both directions. Keep every file
faint-friendly: the game draws them at low opacity over the terrain, and the
map must stay readable underneath.

## What to deliver (`art/source/A14/map/weather/`)

| File | Canvas | Frames | Design |
|---|---|---|---|
| `rain.png` | 512×512 | 1 | Light rain: thin, broken streaks **slanting from the top right down to the bottom left**, about 20° off vertical, 3–6 art pixels long, pale blue-grey with a lighter head. Sparse: most of the canvas stays empty. The game slides it along the slant so the rain falls diagonally. |
| `rain-heavy.png` | 512×512 | 1 | Storm rain: the same slant and colours, about three times as many streaks and longer (6–10 art pixels). |
| `splash.png` | 256×256 per frame | 4 | A raindrop hitting the ground: a tiny splash, then a ring widening and fading. The game scatters it on tiles while it rains. |
| `snow.png` | 512×512 | 1 | Near snow: scattered flakes of 1–2 art pixels, off-white with a faint blue shade on some, irregular spacing (no grid or rows). |
| `snow-far.png` | 512×512 | 1 | Far snow: smaller, fainter and denser flakes (1 art pixel). The game slides it slower than `snow.png`, so the two layers give depth. |
| `fog.png` | 1024×1024 | 1 | Fog: long, soft, horizontal banks and wisps in 2–3 pale grey tones, their edges broken up with a dither so they fade into nothing. About half the canvas covered. It spans 2×2 tiles. |
| `clouds.png` | 1024×1024 | 1 | Cloud shadows: a few large, rounded cloud shapes in one dark slate tone, dithered at their edges, about a third of the canvas covered. The game drifts them slowly across the map on an overcast day. |
| `lightning.png` | 512×512 per frame | 3 | A lightning strike seen from above and a little to the side: a forked white-violet bolt coming down onto the centre of a tile. 1 the full bolt, 2 the bolt flickering thinner, 3 a last branch fading. **No glow or flash painted**; the game flashes the map itself. |

8 files, 13 frames.

## Rules

- **Seamless:** every 1-frame texture must tile with no visible seam or
  repeat line when four copies are placed 2×2. Check the edges carefully:
  a streak or wisp that leaves one edge comes back in on the opposite edge.
- **Grain:** the 8 px art-pixel grain of the terrain tiles (64×64 art
  pixels per 512 px canvas), aligned to the top-left corner; 8 px for the
  1024 px textures too (128×128 art pixels), and for the 256 px splash
  frames (32×32 art pixels).
- **Transparency:** fully transparent or fully opaque pixels only (no
  partial alpha); fade with dither, not with translucency. The game sets
  the opacity.
- **No outline** on rain, snow, fog or clouds (they are weather, not
  objects). The lightning bolt and the splash may have a 1-art-pixel dark
  edge where it helps them read.
- **Animation** (splash and lightning): 1 row of frames with 32 px
  transparent gutters; the object stays in the same place in every frame.
- Colours stay muted and natural, like the rest of the map: no neon blue
  rain or pure white snow.

## Check before delivering

- Each texture tiled 3×3 at full size (no seams, no obvious repeat), and
  laid over a 3×3 patch of forest and road tiles at game size (128 px per
  tile) at about 35% opacity, on day and on a darkened (night) copy.
- `rain.png` slid along its slant over a few frames (a strip is enough):
  it should read as rain falling diagonally.
- The splash and lightning frames as strips at full size and at game size.
