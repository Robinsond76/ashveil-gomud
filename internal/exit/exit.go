package exit

import "github.com/GoMudEngine/GoMud/internal/gamelock"

// There is a magic portal of Chuckles, magic portal of Henry here!
// There is a magical hole in the east wall here!
type TemporaryRoomExit struct {
	RoomId       int    // Where does it lead to?
	Title        string // Does this exit have a special title?
	UserId       int    // Who created it?
	SpawnedRound uint64 `yaml:"-"` // When the temp exit was created
	Expires      string // When will it be auto-cleaned up?
}

type RoomExit struct {
	RoomId        int
	Secret        bool          `yaml:"secret,omitempty"`
	MapDirection  string        `yaml:"mapdirection,omitempty"`   // Optionaly indicate the direction of this exit for mapping purposes
	ExitMessage   string        `yaml:"exitmessage,omitempty"`    // If set, this message is sent to the user, followed by a delay, before they actually go through the exit.
	Lock          gamelock.Lock `yaml:"lock,omitempty"`           // 0 - no lock. greater than zero = difficulty to unlock.
	TravelProfile string        `yaml:"travel_profile,omitempty"` // Optional expedition travel profile name. Empty preserves instant movement.
}

func (re RoomExit) HasLock() bool {
	return re.Lock.Difficulty > 0
}

var DirectionDeltas = map[string][3]int{
	"north":     {0, -1, 0},
	"south":     {0, 1, 0},
	"west":      {-1, 0, 0},
	"east":      {1, 0, 0},
	"northwest": {-1, -1, 0},
	"northeast": {1, -1, 0},
	"southwest": {-1, 1, 0},
	"southeast": {1, 1, 0},
	"down":      {0, 0, -1},
	"up":        {0, 0, 1},

	"north-x2":     {0, -2, 0},
	"south-x2":     {0, 2, 0},
	"west-x2":      {-2, 0, 0},
	"east-x2":      {2, 0, 0},
	"northwest-x2": {-2, -2, 0},
	"northeast-x2": {2, -2, 0},
	"southwest-x2": {-2, 2, 0},
	"southeast-x2": {2, 2, 0},

	"north-x3":     {0, -3, 0},
	"south-x3":     {0, 3, 0},
	"west-x3":      {-3, 0, 0},
	"east-x3":      {3, 0, 0},
	"northwest-x3": {-3, -3, 0},
	"northeast-x3": {3, -3, 0},
	"southwest-x3": {-3, 3, 0},
	"southeast-x3": {3, 3, 0},

	"north-gap":     {0, -1, 0},
	"south-gap":     {0, 1, 0},
	"west-gap":      {-1, 0, 0},
	"east-gap":      {1, 0, 0},
	"northwest-gap": {-1, -1, 0},
	"northeast-gap": {1, -1, 0},
	"southwest-gap": {-1, 1, 0},
	"southeast-gap": {1, 1, 0},

	"north-gap2":     {0, -2, 0},
	"south-gap2":     {0, 2, 0},
	"west-gap2":      {-2, 0, 0},
	"east-gap2":      {2, 0, 0},
	"northwest-gap2": {-2, -2, 0},
	"northeast-gap2": {2, -2, 0},
	"southwest-gap2": {-2, 2, 0},
	"southeast-gap2": {2, 2, 0},

	"north-gap3":     {0, -3, 0},
	"south-gap3":     {0, 3, 0},
	"west-gap3":      {-3, 0, 0},
	"east-gap3":      {3, 0, 0},
	"northwest-gap3": {-3, -3, 0},
	"northeast-gap3": {3, -3, 0},
	"southwest-gap3": {-3, 3, 0},
	"southeast-gap3": {3, 3, 0},
}

var compassDirections = map[string]struct{}{
	"north":     {},
	"south":     {},
	"west":      {},
	"east":      {},
	"northwest": {},
	"northeast": {},
	"southwest": {},
	"southeast": {},
	"down":      {},
	"up":        {},
}

func GetDelta(exitName string) (x, y, z int) {
	if delta, ok := DirectionDeltas[exitName]; ok {
		return delta[0], delta[1], delta[2]
	}
	return 0, 0, 0
}

func IsDirectionalExit(exitName string) bool {
	_, ok := DirectionDeltas[exitName]
	return ok
}

func IsCompassDirection(exitName string) bool {
	_, ok := compassDirections[exitName]
	return ok
}
