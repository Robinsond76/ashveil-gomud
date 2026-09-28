package camping

import (
	"reflect"
	"testing"
)

func TestCampLines(t *testing.T) {
	names := map[int]string{7: "Dain", 9: "Mira"}
	nameOf := func(id int) string { return names[id] }
	cases := []struct {
		name   string
		camps  []RoomCamp
		viewer int
		want   []string
	}{
		{"none", nil, 7, []string{}},
		{"own, unlit", []RoomCamp{{LeaderUserID: 7}}, 7, []string{"A camp is pitched here, around a cold fire pit."}},
		{"own, lit", []RoomCamp{{LeaderUserID: 7, FireLit: true}}, 7, []string{"A camp is pitched here: bedrolls around a crackling campfire."}},
		{"someone else's", []RoomCamp{{LeaderUserID: 7, FireLit: true}}, 9, []string{"Dain's camp is pitched here: bedrolls around a crackling campfire."}},
		{"unknown leader", []RoomCamp{{LeaderUserID: 3}}, 9, []string{"A camp is pitched here, around a cold fire pit."}},
		{"two", []RoomCamp{{LeaderUserID: 7}, {LeaderUserID: 9, FireLit: true}}, 9, []string{
			"Dain's camp is pitched here, around a cold fire pit.",
			"A camp is pitched here: bedrolls around a crackling campfire.",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CampLines(tc.camps, tc.viewer, nameOf); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("CampLines = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoomCampsWithoutReader(t *testing.T) {
	SetRoomCampsReader(nil)
	if got := RoomCamps(1); got != nil {
		t.Fatalf("RoomCamps without a reader = %v", got)
	}
	SetRoomCampsReader(func(roomID int) []RoomCamp { return []RoomCamp{{LeaderUserID: roomID}} })
	t.Cleanup(func() { SetRoomCampsReader(nil) })
	if got := RoomCamps(5); len(got) != 1 || got[0].LeaderUserID != 5 {
		t.Fatalf("RoomCamps = %v", got)
	}
}
